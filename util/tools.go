package util

import (
	"fmt"
	"strconv"
)

const Msglen_tovnetidsize = Msglensize + Msgvnetidsize
const Msglen_toconnidsize = Msglensize + Msgvnetidsize + Msgconnidsize
const Msglen_toctrlsize = Msglensize + Msgvnetidsize + Msgconnidsize + Msgctrlsize
const Msglen_toserveidsize = Msglensize + Msgvnetidsize + Msgconnidsize + Msgctrlsize + Msgserveidsize

// 将数据包中，头部几个标识位解析出来，输出VnetMessage
func Parsemsg_vnet(msg []byte) (VnetMessage, bool) {
	if len(msg) < Msglen_toserveidsize {
		return VnetMessage{}, false
	}
	return VnetMessage{
		Vnetconnid: string(msg[Msglensize:Msglen_toconnidsize]),
		Vnetid:     string(msg[Msglensize:Msglen_tovnetidsize]),
		Connid:     string(msg[Msglen_tovnetidsize:Msglen_toconnidsize]),
		Ctrl:       string(msg[Msglen_toconnidsize:Msglen_toctrlsize]),
		Serveid:    string(msg[Msglen_toctrlsize:Msglen_toserveidsize]),
		Msg:        msg,
	}, true
}

// 将数据包中，头部几个标识位解析出来，输出LocalMessage
func Parsemsg_local(msg []byte) (LocalMessage, bool) {
	if len(msg) < Msglen_toserveidsize {
		return LocalMessage{}, false
	}
	connid, err := strconv.Atoi(string(msg[Msglen_tovnetidsize:Msglen_toconnidsize]))
	if err != nil {
		fmt.Println("read msg connid err:", err)
		return LocalMessage{}, false
	}
	return LocalMessage{
		Vnetconnid: string(msg[Msglensize:Msglen_toconnidsize]),
		Vnetid:     string(msg[Msglensize:Msglen_tovnetidsize]),
		Connid:     connid,
		Ctrl:       string(msg[Msglen_toconnidsize:Msglen_toctrlsize]),
		Serveid:    string(msg[Msglen_toctrlsize:Msglen_toserveidsize]),
		Msg:        msg[Msglen_toserveidsize:],
	}, true
}

// 为数据包增加头部几个标识位，输出带头的[]byte
// func Packmsg_local(localmsg LocalMessage) []byte

// 产生一个不在序列中的id，序列i从1开始，最大为max
func Generateid(old map[int]struct{}, max int) (int, bool) {
	for i := 1; i <= max; i++ {
		if _, ok := old[i]; !ok {
			return i, true
		}
	}
	return -1, false
}

// 产生一个不在序列中的id，序列i从1开始，最大为max // 输入是any，但仅限int
func Generateid_sm_(old []any, max int) (int, bool) {
	var oldmap = make(map[int]struct{})
	for _, o := range old {
		oldmap[o.(int)] = struct{}{}
	}
	return Generateid(oldmap, max)
}

// 将int转为固定长度string
func Fixinttostr(i int, l int) string {
	is := strconv.Itoa(i)
	is = "0000000000"[:l-len(is)] + is
	return is
}

// 为数据包增加msglensize字节流长度标识
func Addmsglen[T string | []byte](msg_ T) []byte {
	msg := []byte(msg_)
	msglen := len(msg) + Msglensize
	return append([]byte(Fixinttostr(msglen, Msglensize)), msg...)
}
