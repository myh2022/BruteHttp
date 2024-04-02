package main

import (
	"BruteHttp/HttpTask"
	"BruteHttp/JsTask"
	"BruteHttp/global"
	"BruteHttp/ipTask"
	"BruteHttp/routers"
	"fmt"
	"sync"
)

/*
	type Options struct {
	FingerDetect      bool
	CaptureScreenshot bool
	JsDetect          bool
	Async             bool
}
*/

func Task() {

}

func Web() {

}

var initWg sync.WaitGroup

// 包装成一个controller去调用
func TaskController() {

	initWg.Add(1)
	var shareDataChan = make(chan global.ShareData)
	//var ipChan = make(chan global.IpPackage)
	var wg sync.WaitGroup
	targetList := []string{"baidu.com"}
	httpOptions := HttpTask.Options{true, false, false, false, targetList, "test"}
	ipOptions := ipTask.Options{"normal"}
	initWg.Done()
	initWg.Wait()

	wg.Add(2)
	go func() {
		defer wg.Done()
		httpOptions.HttpTask(shareDataChan)
	}()

	go func() {
		defer wg.Done()
		ipOptions.IpTask(shareDataChan)
	}()
	wg.Wait()
	// 开始js扫描任务，从数据库取数据
	JsTask.JsTask()

	fmt.Println("done")
}

func main() {

	r := routers.Router()

	r.Run(":8081")
}
