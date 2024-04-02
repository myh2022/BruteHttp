package JsTask

import (
	"crypto/tls"
	"fmt"
	"github.com/gocolly/colly"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type JsData struct {
	Id           int
	Site         string
	RelativeSite string
	Info         map[string]string
	Time         string
	TaskName     string
}

type Rule struct {
	ID        string `yaml:"id"`
	Enabled   bool   `yaml:"enabled"`
	Pattern   string `yaml:"pattern"`
	Content   string `yaml:"content,omitempty"`
	Target    string `yaml:"target,omitempty"`
	Source    string `yaml:"source,omitempty"`
	SourceTag string `yaml:"source_tag,omitempty"`
}

type Config struct {
	Rules        []Rule `yaml:"rules"`
	ExcludeRules []Rule `yaml:"exclude_rules"`
}

var getInfoDone = make(chan bool)
var detectWorkerDone = make(chan JsData)

func initPattern() Config {
	yamlFile, err := ioutil.ReadFile("JsTask/pattern.yaml")
	if err != nil {
		fmt.Println(err)
	}
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Println(err)
	}

	return config
}

// 从数据库中取出返回值为200，且taskName=taskName的值，作为js探测目标
func createTargetList(taskName string) {

}

func (config *Config) getResponseBodyInfo(body string, jsData *JsData) {
	infoMap := make(map[string]string)

	for _, rule := range config.Rules {
		pattern := rule.Pattern
		re, err := regexp.Compile(pattern)

		if err != nil {
			fmt.Println(err)
		}
		matchStrings := re.FindAllString(body, -1)
		if matchStrings == nil {
			continue
		}
		infoMap[rule.ID] = strings.Join(matchStrings, ",")
	}
	jsData.Info = infoMap
	getInfoDone <- true
}

func (config *Config) jsCollyStart(url string) {
	requestUrl := "http://" + url
	var result JsData
	c := colly.NewCollector(
		colly.AllowedDomains(url),
		colly.Async(false),
		colly.IgnoreRobotsTxt(),
	)

	c.WithTransport(&http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("send")
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("recv")
		if r.StatusCode != 200 {
			return
		}
		result.Site = requestUrl
		result.RelativeSite = requestUrl
		go config.getResponseBodyInfo(string(r.Body), &result)
	})

	c.OnHTML("script", func(e *colly.HTMLElement) {
		attr := e.Attr("src")
		if attr == "" {
			return
		}
		if strings.Contains(attr, "https://") || strings.Contains(attr, "http://") {
			c.Visit(attr)
		} else {
			jsUrl := requestUrl + "/" + attr
			c.Visit(jsUrl)
		}

	})

	c.OnError(func(r *colly.Response, err error) {

	})

	c.OnScraped(func(r *colly.Response) {
		// 发送信号量
		<-getInfoDone
		detectWorkerDone <- result
		//fmt.Println(result)
		fmt.Println("js done")
	})
	c.Visit(requestUrl)
}

var initWg = sync.WaitGroup{}

func JsTask() {
	initWg.Add(1)
	startTime := time.Now()
	config := initPattern()
	var urlList = []string{"www.baidu.com"}
	var resultList []JsData
	var ch = make(chan struct{}, 16)
	var allDone = make(chan bool)
	var wg = sync.WaitGroup{}
	initWg.Done()
	initWg.Wait()
	fmt.Println("初始化完成-----")

	go func() {
		for _, url := range urlList {
			ch <- struct{}{}
			wg.Add(1)
			go func(url string) {
				config.jsCollyStart(url)
				defer wg.Done()
				<-ch
			}(url)
		}
		wg.Wait()
		allDone <- true
	}()

	for {
		select {
		case jsData := <-detectWorkerDone:
			resultList = append(resultList, jsData)
		case <-allDone:
			fmt.Println(resultList)
			fmt.Println(time.Since(startTime))
			return
		}
	}

}
