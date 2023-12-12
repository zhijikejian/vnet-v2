package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"util"
)

var vnetids_sm_ = util.NewSyncMap()   // map[int]struct{} // vnet连接映射的id列表
var vnetconns_sm_ = util.NewSyncMap() // map[string]net.Conn // vnet连接映射，{vnetid:net.Conn}
var vnetlist_sm_ = util.NewSyncMap()  // map[string]map[string]string // vnet列表，{vnetid:{name:name,group:group}}
var serveids_sm_ = util.NewSyncMap()  // map[int]struct{} // vnet申请的serve的id列表
var servelist_sm_ = util.NewSyncMap() // map[string]map[string]util.Serve // 保存的所有vnet的服务信息，{vnetid:{serveid:util.Serve}}
var servemap_sm_ = util.NewSyncMap()  // map[string]string // 服务路由，{serveid:vnetid}
var hearttime_sm_ = util.NewSyncMap() // map[string]time.Second // 记录各客户端心跳时间，{vnetid:int64}

var vnetconns_sum = 0

var lastvnetmsgtime_sm_ = util.NewSyncMap() // map[string]time.Now().UnixNano() // vnetmsg的最新接收时间，用来做超时判断
func updatevnetmsgtime(vnetid string) {
	lastvnetmsgtime_sm_.Store(vnetid, time.Now().UnixNano())
}

const timeinterval = time.Second / 10 // timeout一次100ms

func init() {
	if len(os.Args) >= 2 {
		util.Configfile = os.Args[len(os.Args)-1]
	}
	fmt.Println("reading", util.Configfile, "...")
	if !util.Loadconfig() {
		fmt.Println(util.Configfile, "has err")
		os.Exit(0)
	}
}

func main() {
	fmt.Println(time.Now().Format("[2006-01-02 15:04:05]"), "route start ...")
	listen, err := net.Listen("tcp", util.Routeaddr.Ip+":"+util.Routeaddr.Port)
	if err != nil {
		fmt.Println("route listen err:", err)
		return
	}
	defer listen.Close()
	fmt.Println("route listening", util.Routeaddr.Ip+":"+util.Routeaddr.Port, "...\n")
	go lastvnetmsgcheck() // 心跳超时检测 定时检测 // 改为最后消息超时检测
	go func() {           // 没用的东西，只是可以在终端输入空格
		var reader = bufio.NewReader(os.Stdin)
		for {
			reader.ReadString('\n')
		}
	}()
	// 建立连接
	for {
		vnetconn, err := listen.Accept()
		if err != nil {
			fmt.Println("route accept vnetconn err:", err)
			break
		}
		go waitconn(vnetconn)
	}
	fmt.Println("\nroute exit ---")
	os.Exit(0)
}

// 获取一个vnetconn，收听vnet端循环发送的连接类型，config/work，并做出分类和反应
func waitconn(vnetconn net.Conn) {
	var id = 0
	var workname string = ""
	var configname string = ""
	// 接收数据包
	for {
		var buffer [util.Buffersize]byte
		bufsize, err := vnetconn.Read(buffer[:])
		if err != nil {
			break
		}
		bufmsg := string(buffer[:bufsize])
		if strings.HasPrefix(bufmsg, "config") {
			if !strings.HasPrefix(bufmsg, "config/ok") {
				// 检查数量限制
				if vnetconns_sm_.Len() > util.Vnetslimit {
					break
				}
				// 配置连接
				if id_, ok := util.Generateid_sm_(vnetids_sm_.Keys(), util.Vnetslimit); ok {
					// 返回确认+vnetid
					id = id_
					vnetids_sm_.Store(id, struct{}{}) // 更新 vnetids // 立刻更新
					vnetid := util.Fixinttostr(id, util.Msgvnetidsize)
					_, err := vnetconn.Write([]byte("config/ok/" + vnetid))
					if err != nil {
						fmt.Println("vnetconn write config/ok err:", err)
						break
					}
					configname = bufmsg[7:] // string(buffer[7:bufsize])
				} else {
					break
				}
			} else {
				// 确认配置连接
				// 将连接转入 confighandle
				configconn(id, configname, vnetconn)
				// 处理完成一定要退出
				break
			}
		}
		if strings.HasPrefix(bufmsg, "work") {
			// 工作连接
			if !strings.HasPrefix(bufmsg, "work/ok") {
				// 返回确认
				_, err := vnetconn.Write([]byte("work/ok"))
				if err != nil {
					fmt.Println("vnetconn write work/ok err:", err)
					break
				}
				workname = bufmsg[5:] // string(buffer[5:bufsize])
			} else {
				// 将连接转入 workhandle
				workconn(workname, vnetconn)
				// 处理完成一定要退出
				// break
				return // 使用return直接退出避免关闭vnetconn
			}
		}
	}
	vnetconn.Close()
	// 附加一个同步vnetids
	syncvnetids()
}

// work相关 -------------------------------------------------------------------------
// 连接绑定相关逻辑
func workconn(workname string, vnetconn net.Conn) {
	// 检查workinfo
	workinfo := strings.Split(workname, "/")
	if len(workinfo) < 5 {
		fmt.Println("workname err", workname)
		return
	}
	// 数据结构
	var wc = WorkConn{
		Name:       workinfo[0],
		Vnetid:     workinfo[1],
		ToVnetid:   workinfo[2],
		Vnetconnid: workinfo[3],
		Serveid:    workinfo[4],
		Req:        nil,
		Resp:       nil,
		Ready:      false,
		Work:       false,
	}
	// 识别和添加vnetconn
	if wc.Vnetid == util.Msgvnetidempty || wc.ToVnetid == util.Msgvnetidempty || wc.Vnetid == wc.ToVnetid {
		fmt.Println("vnetid err", wc.Vnetid, wc.ToVnetid)
		return
	}
	if strings.HasPrefix(wc.Vnetconnid, wc.Vnetid) {
		// 是服务请求方
		wc.Req = vnetconn
		// 请求对方服务 // 不在这里做，在配置连接中做 // 就在这里搞
		confighandle(wc.Vnetid, vnetconn, util.Addmsglen(wc.Vnetconnid+util.Ctrlworkconn+wc.Serveid+wc.ToVnetid)) // 这样写兼容后面想改成让配置中连接的模式
	} else if strings.HasPrefix(wc.Vnetconnid, wc.ToVnetid) {
		// 是服务提供方
		wc.Resp = vnetconn
		// 查询原始请求方建立的连接数据结构 // 查询 workconnlist_sm_
		if wc_, ok := workconnlist_sm_.Load(wc.Vnetconnid); ok {
			wc_ := wc_.(WorkConn)
			wc_.Name = wc.Name // 以tovnet提供的为准 // 后续可根据vnetname来做校验
			wc_.Resp = wc.Resp
			// 连接数据信息是否一致
			if wc.Vnetid == wc_.ToVnetid && wc.ToVnetid == wc_.Vnetid && wc.Serveid == wc_.Serveid {
				// 一致则更新
				wc = wc_
				// 检查状态
				if !wc.Isready() {
					wc.Close()
					return
				}
			} else {
				// 不一致则关闭请求方的连接数据
				wc_.Close()
				return
			}
		} else {
			// 没有请求方，则没意义
			return
		}
	}
	// 检查状态
	if wc.Ready {
		// 开启服务
		if !wc.Open() {
			wc.Close()
			return
		}
	}
	// 保存入工作区
	workconnlist_sm_.Store(wc.Vnetconnid, wc)
}

// 连接数据结构
type WorkConn struct {
	Name       string   // workname // 最终以tovnet提供的为准 // 后续可根据vnetname来做校验
	Vnetid     string   // 请求服务vnet
	ToVnetid   string   // 服务提供tovnet
	Vnetconnid string   // vnetconnid
	Serveid    string   // serveid
	Req        net.Conn // vnetconn
	Resp       net.Conn // tovnetconn
	Ready      bool     // 两个连接都具有则true
	Work       bool     // 是否已处于工作状态
}

var workconnlist_sm_ = util.NewSyncMap() // map[string]WorkConn // 工作连接映射 {vnetconnid:workconn}

// 查询是否准备就绪
func (wc *WorkConn) Isready() bool {
	if wc.Req != nil && wc.Resp != nil {
		wc.Ready = true
	} else {
		wc.Ready = false
		wc.Work = false
	}
	return wc.Ready
}

// 开启工作连接
func (wc *WorkConn) Open() bool {
	if !wc.Work {
		if wc.Ready {
			go wc.Workconnbind()
			wc.Work = true
		} else {
			wc.Work = false
		}
	}
	return wc.Work
}

// 建立工作连接
func (wc *WorkConn) Workconnbind() {
	vnetconns_sum += 1
	wg := sync.WaitGroup{}
	var bindconn = func(readconn, writeconn net.Conn, middleware func([]byte) []byte) {
		defer readconn.Close()
		defer writeconn.Close()
		defer wg.Done()
		for {
			// read
			var buffer [util.Buffersize]byte
			bufsize, readerr := readconn.Read(buffer[:])
			if readerr != nil {
				break
			}
			// middleware
			msg := middleware(buffer[:bufsize]) // msg := buffer[:bufsize]
			// write
			_, writeerr := writeconn.Write(msg)
			if writeerr != nil {
				break
			}
		}
	}
	wg.Add(2)
	go bindconn(wc.Req, wc.Resp, decode)
	go bindconn(wc.Resp, wc.Req, encode)
	wg.Wait()
	wc.Close()
	vnetconns_sum -= 1
}

func (wc *WorkConn) Close() {
	wc.Ready = false
	wc.Work = false
	if wc.Req != nil {
		wc.Req.Close()
	}
	if wc.Resp != nil {
		wc.Resp.Close()
	}
	// 需要做一些清理工作
	workconnlist_sm_.Delete(wc.Vnetconnid)
}

// 字节码加密
func encode(msg []byte) []byte {
	return msg
}

// 字节码解密
func decode(msg []byte) []byte {
	return msg
}

// config相关 -------------------------------------------------------------------------
// 配置消息处理相关逻辑
func configconn(id int, configname string, vnetconn net.Conn) {
	defer vnetconn.Close()
	configinfo := strings.Split(configname, "/")
	if len(configinfo) < 2 {
		fmt.Println("configname err", configname)
		return
	}
	// 前置步骤
	vnetids_sm_.Store(id, struct{}{}) // 更新 vnetids
	vnetid := util.Fixinttostr(id, util.Msgvnetidsize)
	vnetconns_sm_.Store(vnetid, vnetconn)                                                        // 更新 vnetconns
	vnetlist_sm_.Store(vnetid, map[string]string{"name": configinfo[0], "group": configinfo[1]}) // 更新 vnetlist
	servelist_sm_.Store(vnetid, map[string]util.Serve{})                                         // 更新 servelist
	hearttime_sm_.Store(vnetid, int64(time.Now().Unix()))                                        // 心跳初始计时
	go allsendserves()                                                                           // 广播serves // 上线广播
	fmt.Println(time.Now().Format("[2006-01-02 15:04:05]"), "vnets+:", vnetconns_sm_.Len(), displayvnets())
	// 读取消息
	var messagecache []byte
	var messagelen int = -1
	for {
		var buffer [util.Buffersize]byte
		bufsize, err := vnetconn.Read(buffer[:])
		if err != nil {
			break
		}
		// 更新vnetmsg的最新接收时间
		updatevnetmsgtime(vnetid)
		// 追加到缓存
		messagecache = append(messagecache, buffer[:bufsize]...)
		// 检查缓存和发送，当缓存长度少于msglen则等待下一条msg
		for {
			// 判断msglen，<0则更新
			if messagelen < 0 {
				if len(messagecache) >= util.Msglensize { // 缓存长度大于msglensize，则更新，否则break
					messagelenint, err := strconv.Atoi(string(messagecache[:util.Msglensize]))
					if err != nil {
						// 转int失败的话，则表示是错误msg，需做错误处理
						messagecache = []byte{} // 清空缓存
						messagelen = -1         // 重置msglen，貌似不需要
						break                   // 退出缓存循环
					} else {
						messagelen = messagelenint // 正确保存msglen
					}
				} else { // 缓存长度<msglensize，等待下一次msg
					break
				}
			}
			// 发送缓存，有msglen，且缓存长度大于msglen
			if len(messagecache) >= messagelen {
				// 发送缓存
				confighandle(vnetid, vnetconn, messagecache[:messagelen])     // 处理这条配置消息
				messagecache = append([]byte{}, messagecache[messagelen:]...) // 更新缓存
				messagelen = -1                                               // 重置msglen
			} else { // 缓存长度<msglen，等待下一次msg
				break
			}
		}
	}
	// 断链后的清理和维护 --------------
	vnetids_sm_.Delete(id)
	vnetconns_sm_.Delete(vnetid)
	fmt.Println(time.Now().Format("[2006-01-02 15:04:05]"), "vnets-:", vnetconns_sm_.Len(), displayvnets())
	vnetlist_sm_.Delete(vnetid)  // 当有一方vent掉线，则应当更新 vnetlist
	hearttime_sm_.Delete(vnetid) // 当有一方vent掉线，则应当结束心跳计时
	// 当有一方vent掉线，关闭这个 vnetid 下的所有连接绑定
	for _, kv := range workconnlist_sm_.KVlist() {
		_, wc := kv[0].(string), kv[1].(WorkConn)
		if wc.Vnetid == vnetid || wc.ToVnetid == vnetid {
			wc.Close()
		}
	}
	// 当有一方vent掉线，关闭这个 vnetid 下的所有 serveid
	if serves, ok := servelist_sm_.Load(vnetid); ok {
		for _, serve := range serves.(map[string]util.Serve) {
			servemap_sm_.Delete(serve.Serveid)
		}
	}
	servelist_sm_.Delete(vnetid) // 更新servelist
	go allsendserves()           // 广播serves // 掉线广播
}

// 处理vnet的配置类消息
func confighandle(vnetid string, vnetconn net.Conn, configmsg []byte) {
	if message, ok := util.Parsemsg_vnet(configmsg); ok {
		switch message.Ctrl {
		case util.Ctrlsetvnetid:
			{
				// vnet获取自己的vnetid
				if !writemsg(util.Addmsglen(vnetid+util.Msgconnidempty+util.Ctrlsetvnetid+util.Msgserveidempty), vnetconn) {
					fmt.Println("Ctrlsetvnetid err", vnetid)
				}
			}
		case util.Ctrlgetserveid:
			{
				// 生成一个和已注册的服务的serveid不重复的id
				id := 0
				if id_, ok := util.Generateid_sm_(serveids_sm_.Keys(), util.Serveslimit); ok {
					id = id_
				} else {
					fmt.Println("Ctrlgetserveid serveid err already full", vnetid)
				}
				serveid := util.Fixinttostr(id, util.Msgserveidsize)
				writemsg(util.Addmsglen(vnetid+util.Msgconnidempty+util.Ctrlgetserveid+serveid), vnetconn)
				if serveid != util.Msgserveidempty {
					serveids_sm_.Store(id, struct{}{}) // 加入serveids
					go func() {
						// 20s 后还没有注册这个id，则删除
						time.Sleep(timeinterval * 200) // 20s
						if _, ok := servemap_sm_.Load(serveid); !ok {
							serveids_sm_.Delete(id)
						}
					}()
				}
			}
		case util.Ctrlregserveid:
			{
				// vnet上传自己的serve，route进行保存servelist，并生成servemap映射
				var allsendservesflag = false // 是否需要广播服务 // 不成功则不广播
				var serve = util.Serve{}
				err := json.Unmarshal(configmsg[util.Msglen_toserveidsize:], &serve)
				if err != nil {
					fmt.Println("Ctrlregserveid json.Unmarshal Serve err", err)
					break
				}
				// 检查serveid是否已存在，主要在servemap中查
				var serveidres = util.Msgserveidempty + serve.Serveid
				if _, ok := servemap_sm_.Load(serve.Serveid); !ok { // 不在servemap
					if _, ok := servelist_sm_.Load(vnetid); !ok { // servelist没有vnetid
						servelist_sm_.Store(vnetid, map[string]util.Serve{})
					}
					if vnetservelist, ok := servelist_sm_.Load(vnetid); ok {
						vnetservelist := vnetservelist.(map[string]util.Serve)
						vnetservelist[serve.Serveid] = serve // 保存入servelist
						servelist_sm_.Store(vnetid, vnetservelist)
					}
					servemap_sm_.Store(serve.Serveid, vnetid) // 添加servemap
					serveidres = serve.Serveid                // 如果成功，则serveid != "000"
					allsendservesflag = true                  // 需要广播服务
				}
				// 回传消息
				if !writemsg(util.Addmsglen(vnetid+util.Msgconnidempty+util.Ctrlregserveid+serveidres), vnetconn) {
					fmt.Println("Ctrlsetvnetid err", vnetid)
				}
				// 广播服务变更
				if allsendservesflag {
					go allsendserves()
				}
			}
		case util.Ctrldelserveid:
			{
				// vnet删除自己的serve，route进行删除servelist，并删除servemap映射
				var allsendservesflag = false // 是否需要广播服务 // 不成功则不广播
				var msgback = vnetid + util.Msgconnidempty + util.Ctrldelserveid + util.Msgserveidempty + message.Serveid
				if _, ok := servelist_sm_.Load(vnetid); ok {
					if vnetservelist, ok := servelist_sm_.Load(vnetid); ok {
						vnetservelist := vnetservelist.(map[string]util.Serve)
						delete(vnetservelist, message.Serveid)
						servelist_sm_.Store(vnetid, vnetservelist)
					}
					servemap_sm_.Delete(message.Serveid)
					msgback = vnetid + util.Msgconnidempty + util.Ctrldelserveid + message.Serveid
					allsendservesflag = true // 需要广播服务
				}
				// 中断所有的serveid提供的链路 // vnet端进行
				// 回传消息
				if !writemsg(util.Addmsglen(msgback), vnetconn) {
					fmt.Println("Ctrldelserveid err", vnetid)
				}
				// 广播服务变更
				if allsendservesflag {
					go allsendserves()
				}
			}
		case util.Ctrlgetserves:
			{
				// 将servelist整理成列表 // 或传输jsonstr或inistr到vnet端再解析为列表展示
				var servelist_ = map[string]map[string]util.Serve{}
				// 检查是否同组
				if vnetinfo_, ok := vnetlist_sm_.Load(vnetid); ok {
					if vnetgroup, ok := vnetinfo_.(map[string]string)["group"]; ok {
						for _, kv := range vnetlist_sm_.KVlist() {
							vnetid, vnetinfo := kv[0].(string), kv[1].(map[string]string)
							if vnetinfo["group"] == vnetgroup { // 同组
								if vnetservelist, ok := servelist_sm_.Load(vnetid); ok {
									servelist_[vnetid] = vnetservelist.(map[string]util.Serve)
								}
							}
						}
					}
				}
				servesjson, err := json.Marshal(servelist_)
				if err != nil {
					fmt.Println("json.Marshal servelist err", err)
					servesjson = []byte(vnetid + util.Msgconnidempty + util.Ctrlgetserves + util.Msgserveidempty)
				} else {
					servesjson = append([]byte(vnetid+util.Msgconnidempty+util.Ctrlgetserves+util.Msgserveidempty), servesjson...)
				}
				if !writemsg(util.Addmsglen(servesjson), vnetconn) {
					fmt.Println("Ctrlgetserves err", vnetid)
				}
			}
		case util.Ctrlheartbeat:
			{
				// 原样回传
				if !writemsg(configmsg, vnetconn) {
					fmt.Println("Ctrlheartbeat err", vnetid)
				}
				// 心跳更新计时
				hearttime_sm_.Store(vnetid, int64(time.Now().Unix()))
			}
		case util.Ctrlsendmsg_noserve:
			{
				// 无注册服务通信，将serveid位当做tovnetid // 不是同组不转发 或者 转发但让对方拒绝？
				var tovnetid = message.Serveid[1:] // vnetid.len=2, serveid.len=3
				// 检查是否同组
				if vnetinfo, ok := vnetlist_sm_.Load(vnetid); ok {
					if tovnetinfo, ok := vnetlist_sm_.Load(tovnetid); ok {
						if vnetgroup, ok := vnetinfo.(map[string]string)["group"]; ok {
							if tovnetgroup, ok := tovnetinfo.(map[string]string)["group"]; ok {
								if vnetgroup != tovnetgroup {
									break // 直接不转发
								}
							}
						}
					}
				}
				if tovnetconn, ok := vnetconns_sm_.Load(tovnetid); ok {
					writemsg(configmsg, tovnetconn.(net.Conn))
				} else {
					fmt.Println("Ctrlsendmsg_noserve tovnetid(serveid) err", tovnetid)
				}
			}
		case util.Ctrlvnetinfo:
			{
				// 将上传的vnet信息更新到vnetlist
				var msgback = vnetid + util.Msgconnidempty + util.Ctrlvnetinfo + util.Msgserveidempty + util.Msgserveidempty
				var vnetinfomap = make(map[string]string)
				err := json.Unmarshal(configmsg[util.Msglen_toserveidsize:], &vnetinfomap)
				if err != nil {
					fmt.Println("json.Unmarshal vnetinfomap err", err)
				} else {
					vnetlist_sm_.Store(vnetid, vnetinfomap)
					msgback = vnetid + util.Msgconnidempty + util.Ctrlvnetinfo + util.Msgserveidempty
				}
				// 回传消息
				if !writemsg(util.Addmsglen(msgback), vnetconn) {
					fmt.Println("Ctrlvnetinfo err", vnetid)
				}
			}
			// 广播服务变更 // 以免在前面服务广播时，vnetlist没有更新，这里
			go allsendserves()
		case util.Ctrlgetvnetsinfo:
			{
				// 将vnetlist整理成列表 // 或传输jsonstr或inistr到vnet端再解析为列表展示
				var vnetlist_ = make(map[string]string)
				// 检查是否同组
				if vnetinfo_, ok := vnetlist_sm_.Load(vnetid); ok {
					if vnetgroup, ok := vnetinfo_.(map[string]string)["group"]; ok {
						for _, kv := range vnetlist_sm_.KVlist() {
							vnetid, vnetinfo := kv[0].(string), kv[1].(map[string]string)
							if vnetinfo["group"] == vnetgroup { // 同组
								vnetlist_[vnetid] = "name:" + vnetinfo["name"] + ", group:" + vnetinfo["group"] // vnetinfo["name"] + vnetinfo["group"] + vnetinfo["other"]
							}
						}
					}
				}
				vnetlist_["all"] = fmt.Sprintf("%d", vnetconns_sum)
				vnetsjson, err := json.Marshal(vnetlist_)
				if err != nil {
					fmt.Println("json.Marshal vnetlist err", err)
					vnetsjson = []byte(vnetid + util.Msgconnidempty + util.Ctrlgetvnetsinfo + util.Msgserveidempty)
				} else {
					vnetsjson = append([]byte(vnetid+util.Msgconnidempty+util.Ctrlgetvnetsinfo+util.Msgserveidempty), vnetsjson...)
				}
				if !writemsg(util.Addmsglen(vnetsjson), vnetconn) {
					fmt.Println("Ctrlgetvnetsinfo err", vnetid)
				}
			}
		case util.Ctrlworkconn:
			{
				// 无注册服务通信，将serveid位当做tovnetid // 不是同组不转发 或者 转发但让对方拒绝？
				var tovnetid = string(configmsg[util.Msglen_toserveidsize:])
				// 检查是否同组
				if vnetinfo, ok := vnetlist_sm_.Load(vnetid); ok {
					if tovnetinfo, ok := vnetlist_sm_.Load(tovnetid); ok {
						if vnetgroup, ok := vnetinfo.(map[string]string)["group"]; ok {
							if tovnetgroup, ok := tovnetinfo.(map[string]string)["group"]; ok {
								if vnetgroup != tovnetgroup {
									break // 直接不转发
								}
							}
						}
					}
				}
				if tovnetconn, ok := vnetconns_sm_.Load(tovnetid); ok {
					writemsg(configmsg, tovnetconn.(net.Conn))
				} else {
					fmt.Println("Ctrlworkconn tovnetid(serveid) err", tovnetid)
				}
			}
		default:
			{
				fmt.Println("switch message.Ctrl err default")
			}
		}
	} else {
		fmt.Println("Parsemsg_vnet message err", string(configmsg))
	}
}

// 复用conn.write代码
func writemsg(msg []byte, conn net.Conn) bool {
	_, err := conn.Write(msg)
	return err == nil
}

// 广播服务变更
func allsendserves() {
	for _, kv := range vnetconns_sm_.KVlist() {
		vnetid, vnetconn := kv[0].(string), kv[1].(net.Conn)
		msg := util.Addmsglen(vnetid + util.Msgconnidempty + util.Ctrlgetserves + util.Msgserveidempty)
		confighandle(vnetid, vnetconn, msg)
	}
	// 附加一个同步serveids
	syncserveids()
}

// 同步serveids
func syncserveids() {
	for _, id := range serveids_sm_.Keys() {
		id := id.(int)
		serveid := util.Fixinttostr(id, util.Msgserveidsize)
		if _, ok := servemap_sm_.Load(serveid); !ok {
			serveids_sm_.Delete(id)
		}
	}
}

// 同步vnetids
func syncvnetids() {
	for _, id := range vnetids_sm_.Keys() {
		id := id.(int)
		vnetid := util.Fixinttostr(id, util.Msgvnetidsize)
		if _, ok := vnetconns_sm_.Load(vnetid); !ok {
			vnetids_sm_.Delete(id)
		}
	}
}

// 格式化显示vnets信息，[[vnetid,vnetaddr],[vnetid,vnetaddr]]
func displayvnets() [][]any {
	var displayvnets = [][]any{}
	for _, kv := range vnetconns_sm_.KVlist() {
		vnetid, vnetconn := kv[0].(string), kv[1].(net.Conn)
		displayvnets = append(displayvnets, []any{vnetid, vnetconn.RemoteAddr().String()})
	}
	return displayvnets
}

// 心跳超时检测 // 检测config.heartbeat // 改为最后消息超时检测，检测lastvnetmsgtime_sm_
func lastvnetmsgcheck() {
	for {
		time.Sleep(time.Second)
		for _, kv := range lastvnetmsgtime_sm_.KVlist() {
			vnetid, hearttime := kv[0].(string), kv[1].(int64)
			if time.Now().Unix()-hearttime > 150 { // 超过 15s 关闭与vnet端的连接
				if vnetconn, ok := vnetconns_sm_.Load(vnetid); ok {
					fmt.Println("vnet", vnetid, "连接超时 ...")
					vnetconn.(net.Conn).Close()
				}
			}
		}
	}
}
