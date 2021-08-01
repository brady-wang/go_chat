package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const ExpireTime = 3600
const MessageSize = 4096
type MyServer struct {
	Ip string
	Port int
	// 在线用户列表
	OnlineMap map[string]*User
	MapLock   sync.RWMutex

	// 消息广播的 chan
	Message chan string
}

func NewServer(ip string ,port int )  *MyServer {
	server := &MyServer{
		Ip:   ip,
		Port: port,
		OnlineMap: make(map[string]*User),
		Message: make(chan string),
	}
	return server
}

// 有用户连接时候的 handler
func (s *MyServer)Handler(conn net.Conn)  {
	// handler
	fmt.Printf("%s 连接成功 \n",conn.RemoteAddr().String())

	// 用户上线 用户加入上线列表 广播当前用户上线消息
	user := NewUser(conn,s)
	user.Online()

	// 监听是否活跃的 chan
	isLive := make(chan bool)

	//接受用户的消息
	go func() {
		buff := make([]byte,MessageSize)
		for {
			n,err := conn.Read(buff)
			if n == 0 {
				user.Offline()
				return
			}

			if err !=nil && err != io.EOF {
				fmt.Println("conn read error",err)
				return
			}
			msg := string(buff[0:n-1])
			isLive <- true
			if msg != "" {
				user.DoMsg(msg)
			}

		}
	}()

	// handler 阻塞
	for  {
		select{
		    case <- isLive: // 重置定时器 什么都不做
			case <- time.After(ExpireTime *time.Second):
			// 已经超时  当前客户端强制关闭
			user.sendMsg("你因超时被踢了\n")
			close(user.C)
			conn.Close()
			return

		}
	}

}

// 监听 message 的 chan一有消息就发送
func (s *MyServer) ListenMessage()  {
	for  {
		msg := <- s.Message
		s.MapLock.Lock()
		for _,cli := range s.OnlineMap{
			cli.C <- msg
		}
		s.MapLock.Unlock()
	}
}

// 广播消息 消息发送给所有的 发送到 chan里面
func (s *MyServer) Broadcast(user *User,msg string)  {
	sendMessage := "["+user.Addr+"]"+user.Name+":"+msg
	fmt.Printf("广播消息 %s \n", sendMessage)
	s.Message <- sendMessage
}

func (s *MyServer) Start()  {
	//socket listen
	listen,err:=net.Listen("tcp",fmt.Sprintf("%s:%d",s.Ip,s.Port))
	if err !=nil{
		fmt.Println("net listen err",err)
		return
	}
	defer listen.Close()

	go s.ListenMessage()

	//accept
	for{
		conn, err := listen.Accept()
		if err !=nil{
			fmt.Println("listen accept error",err)
			continue
		}


		//do handler
		go s.Handler(conn)

	}
	//close
}

