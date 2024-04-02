package ipTask

import (
	"BruteHttp/global"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type IpData struct {
	Id   int
	Port []string
}

type Options struct {
	DictType string
}

var taskCount = 1

// 数组去重
func arrayRemoveDup(ipArray []string) []string {
	var ipMap = make(map[string]bool)
	var resultArray []string
	for _, s := range ipArray {
		if !ipMap[s] {
			ipMap[s] = true
			resultArray = append(resultArray, s)
		}
	}

	return resultArray
}

var portScanResultRecvChan = make(chan IpData)

// 端口扫描
func (options *Options) portScan(ip string) {

	var portList []string
	var openPortList []string
	// 类型初始化
	if options.DictType == NormalPort {
		portList = append(portList, normalPortList...)
		log.Println("[IP TASK] use normal port list")
	} else if options.DictType == Top1000Port {
		log.Println("[IP TASK] use top1000 port list")
	} else if options.DictType == AllPort {
		log.Println("[IP TASK] use all port list")
	}

	for _, port := range portList {
		address := ip + ":" + port
		log.Println("[IP TASK] start scan port: %v, ip: %v", port, ip)
		conn, err := net.DialTimeout("tcp", address, 3*time.Second)

		if err == nil {
			openPortList = append(openPortList, port)
			conn.Close()
		} else {
			continue
		}
	}

	portScanResultRecvChan <- IpData{Port: portList}

}

var recvIpsChan = make(chan []string)

func turnHostToIps(host string) {

	var ipList []string
	ips, err := net.LookupIP(strings.Split(host, `/`)[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, ip := range ips {
		ipList = append(ipList, ip.String())
	}

	recvIpsChan <- ipList

}

const ipPattern = `\d*.\d*.\d*.\d*`

var ipChan = make(chan []string)
var ipScanDoneCh atomic.Value

// 域名变ip

var wg sync.WaitGroup
var turnToIpDone = make(chan bool)

// 配置初始化
func initBaseConfig(workerCount int) (chan struct{}, *sync.WaitGroup, chan bool) {
	var baseWorkerCountChan = make(chan struct{}, workerCount)
	var baseWg sync.WaitGroup
	var baseTaskDoneChan = make(chan bool)

	return baseWorkerCountChan, &baseWg, baseTaskDoneChan
}

func (options *Options) IpTask(shareDataChan chan global.ShareData) {

	wg.Add(1)
	var resultList []IpData
	var hostList []string
	var waitTaskDoneCount = 0
	var removeDupWg sync.WaitGroup
	// 接收到数据
	for {
		shareData := <-shareDataChan

		if shareData.State {
			waitTaskDoneCount++
			hostList = append(hostList, shareData.Data...)
		}

		if waitTaskDoneCount == taskCount {
			break
		}
		// 某个任务完成
	}
	// 域名数组去重
	hostList = arrayRemoveDup(hostList)

	// 域名变ip的参数赋值
	var turnHostToIpChan = make(chan struct{}, 16)
	var turnHostToIpWg sync.WaitGroup
	var turnHostToIpDoneChan = make(chan bool)

	// 端口扫描的参数赋值
	var scanPostChan = make(chan struct{}, 4)
	var scanPostWg sync.WaitGroup
	var scanPostDoneChan = make(chan bool)
	var ipList []string

	var task2Wg sync.WaitGroup
	wg.Done()
	wg.Wait()

	// 将host转换为ip
	task2Wg.Add(2)
	go func() {
		defer task2Wg.Done()
		for _, host := range hostList {
			turnHostToIpChan <- struct{}{}
			turnHostToIpWg.Add(1)
			go func(host string) {
				defer turnHostToIpWg.Done()
				turnHostToIps(host)
				<-turnHostToIpChan
			}(host)
		}
		turnHostToIpWg.Wait()
		turnHostToIpDoneChan <- true
	}()

	// 对ip数组去重
	go func() {
		defer task2Wg.Done()
		for {
			select {
			case data := <-recvIpsChan:
				ipList = append(ipList, data...)
			case <-turnHostToIpDoneChan:
				return
			}
		}
	}()
	task2Wg.Wait()

	removeDupWg.Add(1)
	ipList = arrayRemoveDup(ipList)
	fmt.Printf("[IP TASK] ip list: %v\n", ipList)
	removeDupWg.Done()
	removeDupWg.Wait()

	// 发送任务进行ip端口扫描
	go func() {
		for _, ip := range ipList {
			scanPostWg.Add(1)
			go func(ip string) {
				scanPostChan <- struct{}{}
				defer scanPostWg.Done()
				options.portScan(ip)
				<-scanPostChan
			}(ip)
		}
		scanPostWg.Wait()
		scanPostDoneChan <- true
	}()
	// 对端口扫描情况进行监控
	for {
		select {
		case ipData := <-portScanResultRecvChan:
			resultList = append(resultList, ipData)
		case <-scanPostDoneChan:
			log.Println("[IP TASK] port scan done")
			log.Printf("[IP TASK] result: %v\n", resultList)
			return
		}
	}

}
