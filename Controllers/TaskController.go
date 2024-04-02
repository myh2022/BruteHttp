package Controllers

import (
	"BruteHttp/HttpTask"
	"BruteHttp/JsTask"
	"BruteHttp/global"
	"BruteHttp/ipTask"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"sync"
)

type TaskController struct {
	Ctx    context.Context
	Cancel context.CancelFunc
}

var initWg sync.WaitGroup
var startChan = make(chan bool)
var startTaskWg sync.WaitGroup

func task() {
	initWg.Add(1)
	var shareDataChan = make(chan global.ShareData)
	//var ipChan = make(chan global.IpPackage)
	var wg sync.WaitGroup
	// 从数据库中取任务配置
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

func start() {

	initWg.Add(1)
	var shareDataChan = make(chan global.ShareData)
	//var ipChan = make(chan global.IpPackage)
	var wg sync.WaitGroup
	// 从数据库中取任务配置
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

// 开始任务
func (t *TaskController) StartTask(c *gin.Context) {

	go start()

	fmt.Println("start task")

	c.JSON(http.StatusOK, gin.H{"message": "Task started"})

}

func (t *TaskController) CreateTask(c *gin.Context) {
	fmt.Println("create task")
}

// 用于提前结束扫描任务
func (t *TaskController) CancelTask(c *gin.Context) {

	fmt.Println("结束任务")
	t.Cancel()

}
