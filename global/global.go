package global

import "net"

const WorkerCount = 4

type IpPackage struct {
	IpList []net.IP
	Url    string
}

type ShareData struct {
	Name  string
	Data  []string
	State bool
}
