package vanilla

import (
	"github.com/kfchen81/beego"
	"net"
	"os"
)

var machineInfo = Map{
	"ip": "",
	"hostname": "",
}

//GetMachineInfo 获取机器相关信息，目前包括hostname, ip address
func GetMachineInfo() Map {
	return machineInfo
}

func init() {
	//初始化时获取machine info
	{
		//get hostname
		hostname, error := os.Hostname()
		if error != nil {
			beego.Error(error.Error())
		} else {
			machineInfo["hostname"] = hostname
		}
	}
	
	{
		//get ip address
		conn, error := net.Dial("udp", "114.114.114.114:80")
		if error != nil {
			beego.Error(error.Error())
		} else {
			defer conn.Close()
			ipAddress := conn.LocalAddr().(*net.UDPAddr)
			machineInfo["ip"] = ipAddress.String()
		}
	}
}
