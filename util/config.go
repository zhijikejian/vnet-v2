package util

import (
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

// 数据包的头协议定义
const Buffersize = 1024 * 5 // read缓存容量 // 小于10的Msglensize次方
const Msglensize = 4        // 数据包长度标记容量 // 大于等于Buffersize所需的位数
const Msgvnetidsize = 2     // vnet客户端标记容量
const Msgconnidsize = 3     // vnet客户端conn标记容量
const Msgctrlsize = 2       // 数据包的控制标记容量
const Msgserveidsize = 3    // vnet提供服务的标记容量

// 控制标记 // length应和Msgctrlsize一致
const Ctrlcloseconn = "00"       // 关闭一个vnet的conn
const Ctrlsetvnetid = "01"       // 设置一个vnet的routeid
const Ctrlgetserveid = "02"      // vnet向route申请服务id
const Ctrlregserveid = "03"      // vnet向route注册服务
const Ctrldelserveid = "04"      // vnet向route注销服务
const Ctrlgetserves = "05"       // vnet获取router所有服务
const Ctrlsendmsg = "06"         // 传递消息数据
const Ctrlheartbeat = "07"       // 心跳
const Ctrlsendmsg_noserve = "08" // 请求未注册服务
const Ctrlvnetinfo = "09"        // 上传vnet的一些信息
const Ctrlgetvnetsinfo = "10"    // 获取全部vnet的信息
const Ctrlworkconn = "11"        // 通知对方vnet建立连接

var Msgvnetidempty = Fixinttostr(0, Msgvnetidsize)   // 常用的空的vnetid // "00"
var Msgconnidempty = Fixinttostr(0, Msgconnidsize)   // 常用的空的connid // "00"
var Msgserveidempty = Fixinttostr(0, Msgserveidsize) // 常用的空的serveid // "00"

// 10的size次方
func exponent(size int) int {
	result := 10
	for i := 1; i < size; i++ {
		result = result * 10
	}
	return result
}

// 数量限制 // 由于Msgvnetidsize，Msgserveidsize和Msgconnidsize限制，以及初始不能为0，导致其数量限制
var Vnetslimit = exponent(Msgvnetidsize) - 1 // 99
var Connslimit = exponent(Msgconnidsize) - 1 // 99
// var Ctrlslimit = exponent(Msgctrlsize) - 1 // 99
var Serveslimit_def = 500                                        // 500 // 预留后500个serveid来自定义[500,999]
var Serveslimit = exponent(Msgserveidsize) - 1 - Serveslimit_def // 500 // 预留后500个serveid来自定义[500,999]

// 配置文件
const Defaultconfigfile = "config.ini"

var Configfile = Defaultconfigfile

// route监听地址
type RouteAddr struct {
	Ip   string `ini:"ip"`
	Port string `ini:"port"`
}

var Routeaddr RouteAddr

// vnet信息类型
type VnetInfo struct {
	Name           string `ini:"name"`
	Group          string `ini:"group"`
	Terminal       bool   `ini:"terminal"`
	Startserve     bool   `ini:"startserve"`
	Startuse       bool   `ini:"startuse"`
	Allow_unserve  bool   `ini:"allow_unserve"`
	Unserve_remote bool   `ini:"allow_remote"`
	Unserve_num    int    `ini:"unserve_num"`
}

var Vnetinfo VnetInfo

// 服务类型
type Serve struct {
	Serveid string `ini:"serveid"`
	Type    string `ini:"type"`
	Ip      string `ini:"ip"`
	Port    string `ini:"port"`
	Name    string `ini:"name"`
	Info    string `ini:"info"`
	Active  bool   `ini:"active"`
}
type ServeList []Serve

// 服务列表
var Servelist ServeList

// 映射类型
type Use struct {
	Serveid string `ini:"serveid"`
	Ip      string `ini:"ip"`
	Port    string `ini:"port"`
	Active  bool   `ini:"active"`
}
type UseList []Use

// 映射列表
var Uselist UseList

// 初始化，解析配置文件
func Loadconfig() bool {
	Routeaddr = RouteAddr{}
	Vnetinfo = VnetInfo{}
	Servelist = ServeList{}
	Uselist = UseList{}
	cfg, err := ini.Load(Configfile)
	if err != nil {
		fmt.Println("read", Configfile, "err:", err)
		return false
	}
	// 收集routeaddr,servelist和uselist
	for _, key := range cfg.SectionStrings() {
		if key == "route" {
			var routeaddr = RouteAddr{
				Ip:   "",
				Port: "7749",
			}
			err := cfg.Section(key).MapTo(&routeaddr)
			if err != nil {
				fmt.Println("mapto routeaddr err", err)
			} else {
				Routeaddr = routeaddr
			}
		} else if key == "vnet" {
			var vnetinfo = VnetInfo{
				Name:           "",
				Group:          "default",
				Terminal:       false,
				Startserve:     true,
				Startuse:       true,
				Allow_unserve:  false,
				Unserve_remote: false,
				Unserve_num:    1,
			}
			err := cfg.Section(key).MapTo(&vnetinfo)
			if err != nil {
				fmt.Println("mapto vnetinfo err", err)
			} else {
				if !vnetinfo.Terminal {
					vnetinfo.Startserve = true
					vnetinfo.Startuse = true
					// vnetinfo.Allow_unserve = false
				}
				if len(vnetinfo.Group) <= 0 {
					vnetinfo.Group = "default"
				}
				Vnetinfo = vnetinfo
			}
		} else if strings.HasPrefix(key, "serve.") {
			var serve = Serve{
				Serveid: Msgserveidempty,
				Type:    "tcp",
				Ip:      "127.0.0.1",
				Port:    "",
				Name:    key[6:],
				Info:    "",
				Active:  true,
			}
			err := cfg.Section(key).MapTo(&serve)
			if err != nil {
				fmt.Println("mapto serve err", err)
			} else {
				if len(serve.Serveid) > Msgserveidsize {
					serve.Serveid = Msgserveidempty
				}
				serve.Serveid = "0000000000"[:Msgserveidsize-len(serve.Serveid)] + serve.Serveid
				Servelist = append(Servelist, serve)
			}
		} else if strings.HasPrefix(key, "use.") {
			var use = Use{
				Serveid: Msgserveidempty,
				Ip:      "127.0.0.1",
				Port:    "",
				Active:  true,
			}
			err := cfg.Section(key).MapTo(&use)
			if err != nil {
				fmt.Println("mapto use err", err)
			} else {
				if len(use.Serveid) > Msgserveidsize {
					use.Serveid = Msgserveidempty
				}
				use.Serveid = "0000000000"[:Msgserveidsize-len(use.Serveid)] + use.Serveid
				Uselist = append(Uselist, use)
			}
		}
	}
	if Routeaddr.Ip == "" || Routeaddr.Port == "" {
		return false
	}
	if Vnetinfo.Terminal {
		var newservelist = ServeList{}
		if !Vnetinfo.Startserve {
			for _, serve := range Servelist {
				serve.Active = false
				newservelist = append(newservelist, serve)
			}
			Servelist = newservelist
		}
		var newuselist = UseList{}
		if !Vnetinfo.Startuse {
			for _, use := range Uselist {
				use.Active = false
				newuselist = append(newuselist, use)
			}
			Uselist = newuselist
		}
	}
	return true
}

func main() {
	// 测试
	fmt.Println("Configfile", Configfile)
	Loadconfig()
	fmt.Println("Routeaddr", Routeaddr)
	fmt.Println("Vnetinfo", Vnetinfo)
	fmt.Println("Servelist", Servelist)
	fmt.Println("Uselist", Uselist)
}

// route组件中传输简易消息的类型，类型中附加了消息来源vnetid
type RouteMessage struct {
	Vnetid string // 消息来源vnetid
	Msg    []byte // 不裁剪的带头协议数据
}

// route组件中解析的带头协议msg，字符connid的message类型
type VnetMessage struct {
	Vnetconnid string
	Vnetid     string
	Connid     string
	Ctrl       string
	Serveid    string
	Msg        []byte // 不裁剪的带头协议数据
}

// vnet组件中解析的不带头协议msg，字符connid的int类型
type LocalMessage struct {
	Vnetconnid string
	Vnetid     string
	Connid     int
	Ctrl       string
	Serveid    string
	Msg        []byte // 裁剪的不带头协议数据
}
