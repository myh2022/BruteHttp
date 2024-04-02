package HttpTask

import (
	"BruteHttp/global"
	"BruteHttp/models"
	"BruteHttp/whatweb"
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

//type HttpData struct {
//	Id             int
//	Site           string
//	StatusCode     int
//	Header         string
//	Fingers        string
//	IconPath       string
//	ScreenshotPath string
//	Time           string
//	TaskName       string
//}

type Options struct {
	FingerDetect      bool
	CaptureScreenshot bool
	JsDetect          bool
	Async             bool
	TargetList        []string
	TaskName          string
}

var resultList []models.HttpData

var httpDataChan = make(chan models.HttpData)
var FingerDetectDoneChan = make(chan bool)

func formatPath(url string) string {
	// 转换为小写，提取url地址，.->——_
	return strings.ToLower(strings.Replace(strings.Split(url, `/`)[2], ".", "_", -1))
}

func (options *Options) createBruteList(targetList []string) []string {
	var urlList []string
	file, err := os.Open("HttpTask/domain")
	if err != nil {
		panic(err)
		return nil
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text != "" {
			for _, target := range options.TargetList {
				urlList = append(urlList, text+"."+target)
			}
		}
	}
	return urlList
}

func captureScreenshot(httpData *models.HttpData) {

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 5*time.Second)
	defer cancelTimeout()

	var buf []byte
	_ = chromedp.Run(ctx, fullScreenshot(httpData.Site, 90, &buf))
	if err := os.WriteFile("screenshots/"+strings.ToLower(strings.Replace(strings.Split(httpData.Site, `/`)[2], ".", "_", -1))+".png", buf, 0644); err != nil {
		log.Fatal(err)
	}
	log.Printf("[屏幕截图] wrote %v fullscreenshot", httpData.Site)
	// 修改结构体字段值
	httpData.ScreenshotPath = "screenshots/" + strings.ToLower(strings.Replace(strings.Split(httpData.Site, `/`)[2], ".", "_", -1)) + ".png"

}

func fullScreenshot(urlStr string, quality int, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlStr),
		chromedp.FullScreenshot(res, quality),
	}
}

func getWebIcon(iconUrl string, httpData *models.HttpData) {

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := http.Client{
		Transport: t,
		Timeout:   5 * time.Second,
	}

	resp, err := client.Get(iconUrl)

	if err != nil {
		panic(err)
	}
	iconPath := "icons/" + strings.ToLower(strings.Replace(strings.Split(iconUrl, `/`)[2], ".", "_", -1)) + ".jpg"
	file, err := os.Create(iconPath)

	if err != nil {
		log.Println("create file error", err)
	}
	defer file.Close()
	iconData, err := io.ReadAll(resp.Body)
	_, err = file.Write(iconData)

	if err != nil {
		log.Println("keep file error", err)
	}

	httpData.IconPath = iconPath

}

var wapp, _ = whatweb.Init("whatweb/app.json", true)

func fingerDetect(body string, httpData *models.HttpData) {
	httpData.Fingers = "test"
	data := whatweb.Data{}
	data.Url = httpData.Site
	data.Html = body
	result, _ := wapp.Analyze(&data)

	// 类型断言
	s, ok := result.(string)
	if !ok {
		httpData.Fingers = ""
	}
	httpData.Fingers = s

	FingerDetectDoneChan <- true
}

func (options *Options) collyStart(url string) {

	var result models.HttpData

	requestUrl := "http://" + url

	c := colly.NewCollector(
		colly.AllowedDomains(url),
		colly.Async(options.Async),
		colly.IgnoreRobotsTxt(),
	)

	c.SetRequestTimeout(3 * time.Second)

	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	c.OnRequest(func(r *colly.Request) {
		// 这里可以开始指纹识别了
		log.Println("[发送请求] start send request: ", r.URL)
		//fmt.Println("start send request: ", r.URL)

	})

	// 指纹识别， 对header,site的赋值
	c.OnResponse(func(r *colly.Response) {
		// 开始指纹识别
		go fingerDetect(string(r.Body), &result)
		result.Site = r.Request.URL.String()
		result.StatusCode = r.StatusCode
		headers, err := json.Marshal(r.Headers)
		if err != nil {
			fmt.Println(err)
			result.Header = ""
		}
		result.Header = string(headers)
	})

	// 抓取网站icon，从head中查看link标签，判断href是路径还是url地址
	c.OnHTML("head", func(e *colly.HTMLElement) {
		e.ForEach("link", func(_ int, el *colly.HTMLElement) {
			attr := el.Attr("rel")
			if strings.Compare(attr, "icon") == 0 {
				iconUrl := el.Attr("href")
				log.Println("[icon抓取] icon address: ", iconUrl)
				//fmt.Println("icon address: ", iconUrl)
				if strings.Contains(iconUrl, "https://") || strings.Contains(iconUrl, "http://") {
					getWebIcon(iconUrl, &result)
				} else {
					getWebIcon(requestUrl+"/"+iconUrl, &result)
				}
			}
		})

	})

	// 当前抓取结束，对time赋值
	c.OnScraped(func(r *colly.Response) {
		log.Println("scrape done: ", r.Request.URL)
		result.Time = time.DateTime
		// 检测到指纹识别完成，将数据发送到主协程
		<-FingerDetectDoneChan
		// 发送到主协程
		httpDataChan <- result

	})

	c.OnError(func(response *colly.Response, err error) {
		log.Println("[error] ", err)
		//return
	})

	c.Visit(requestUrl)

}

func loadShareIpData(httpData models.HttpData) {

}

// 网站爆破，指纹扫描需要初始化参数的锁
var wg sync.WaitGroup

// 判断什么时候开始网站截图，网站截图需要最后开始，占用大量CPU
var startCaptureChan = make(chan bool)

// 网站截图需要初始化参数的锁
var capWg sync.WaitGroup

func (options *Options) HttpTask(shareDataChan chan global.ShareData) {
	startTime := time.Now()
	// 加锁防止options未赋值就开始goroutine
	wg.Add(1)
	var urlList = options.createBruteList(options.TargetList)
	ch := make(chan struct{}, 32)
	var waitGroup sync.WaitGroup
	var detectDone = make(chan bool)
	wg.Done()
	wg.Wait()

	// 开启16个线程去进行初步抓取
	go func() {
		for _, url := range urlList {
			ch <- struct{}{}
			waitGroup.Add(1)
			go func(url string) {
				defer waitGroup.Done()
				options.collyStart(url)
				<-ch
			}(url)
		}
		waitGroup.Wait()

		detectDone <- true
	}()

	go func() {
		for {
			select {
			case httpData := <-httpDataChan:
				//go sendIpByHttpTask(httpData, ipChan)
				resultList = append(resultList, httpData)
			case <-detectDone:
				var dataList []string
				fmt.Println("all done")
				for _, data := range resultList {
					dataList = append(dataList, data.Site)
				}
				shareDataChan <- global.ShareData{Name: "httpTask", Data: dataList, State: true}
				startCaptureChan <- true
				return
			}
		}
	}()
	// 开启截图goroutine
	<-startCaptureChan
	// 判断是否开启网站截图功能
	if options.CaptureScreenshot {
		// 开始对网站截图进行初始化
		capWg.Add(1)
		var capCh = make(chan struct{}, 4)
		var capRoutineWg = sync.WaitGroup{}
		capWg.Done()
		capWg.Wait()
		fmt.Println("开始网站截图")
		for i, _ := range resultList {
			capRoutineWg.Add(1)
			capCh <- struct{}{}
			go func() {
				defer capRoutineWg.Done()
				captureScreenshot(&resultList[i])
				<-capCh
			}()
			capRoutineWg.Wait()
		}
	}

	fmt.Println(time.Since(startTime))

	fmt.Println(resultList)

	// 对数据库操作
	err := models.HttpData{}.InsertOrUpdateHttpData(resultList)
	fmt.Println(err)
}
