package main

import (
	"fmt"
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	Conn net.Conn
	//server
	Server *MyServer
}

func NewUser(conn net.Conn, server *MyServer) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		Conn:   conn,
		Server: server,
	}
	go user.ListenMessage()
	return user
}

// 监听当前 user chan 一旦有消息 发送客户端
func (u *User) ListenMessage() {
	for {
		msg := <-u.C
		u.Conn.Write([]byte(msg + "\n"))
	}
}

func (u *User) Online() {
	u.Server.MapLock.Lock()
	u.Server.OnlineMap[u.Name] = u
	u.Server.MapLock.Unlock()

	//广播当前用户上线消息
	u.Server.Broadcast(u, "已上线")

}

func (u *User) Offline() {
	u.Server.MapLock.Lock()
	delete(u.Server.OnlineMap, u.Name)
	u.Server.MapLock.Unlock()

	//广播当前用户上线消息
	u.Server.Broadcast(u, "已下线")
}

func (u *User) DoMsg(msg string) {

	if msg == "who" {
		u.Server.MapLock.Lock()
		for _, v := range u.Server.OnlineMap {
			msg := "[" + v.Addr + "]" + v.Name + " 在线 ...\n"
			u.sendMsg(msg)
		}
		u.Server.MapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]
		if _, ok := u.Server.OnlineMap[newName]; ok {
			u.sendMsg("用户名已被占用")
		} else {
			u.Server.MapLock.Lock()
			delete(u.Server.OnlineMap, u.Name)
			u.Server.OnlineMap[newName] = u
			u.Server.MapLock.Unlock()
			u.Name = newName
			u.sendMsg("你已经成功修改用户名: " + u.Name + "\n")
		}

	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 获取对方用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			u.sendMsg("消息格式不正确 请使用 \"to|张三|你好啊\"格式\n")
			return
		}

		if remoteUser, ok := u.Server.OnlineMap[remoteName]; ok {
			fmt.Println(msg)
			content := strings.Split(msg, "|")[2]
			if content == "" {
				u.sendMsg("无消息内容 请重发\n")
				return
			} else {
				content = u.Name + " 对你说" + content+"\n"
				remoteUser.sendMsg(content)
			}
		} else {
			u.sendMsg("该用户不存在\n")
			return
		}
	} else {
		u.Server.Broadcast(u, msg)

	}

}

func (u *User) sendMsg(msg string) {
	u.Conn.Write([]byte(msg))
}
