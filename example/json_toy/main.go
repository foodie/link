package main

import (
	"log"

	"github.com/funny/link"
	"github.com/funny/link/codec"
)

type AddReq struct {
	A, B int
}

type AddRsp struct {
	C int
}

func main() {
	//解析器类型
	json := codec.Json()
	json.Register(AddReq{})
	json.Register(AddRsp{})

	//获取一个Server，定义处理session的函数
	server, err := link.Listen("tcp", "0.0.0.0:0", json,
		0 /* sync send */, link.HandlerFunc(serverSessionLoop))
	checkErr(err)
	addr := server.Listener().Addr().String()
	go server.Serve()

	//发起请求
	client, err := link.Dial("tcp", addr, json, 0)
	checkErr(err)
	clientSessionLoop(client)
}

//服务端的处理
func serverSessionLoop(session *link.Session) {
	//服务端处理所有的结果
	for {
		//获取请求，通过Codec去处理
		req, err := session.Receive()
		checkErr(err)

		//服务端发送处理结果
		err = session.Send(&AddRsp{
			req.(*AddReq).A + req.(*AddReq).B,
		})
		checkErr(err)
	}
}

//客户端请求的方法
func clientSessionLoop(session *link.Session) {
	for i := 0; i < 10; i++ {
		//发送请求
		err := session.Send(&AddReq{
			i, i,
		})
		checkErr(err)
		log.Printf("Send: %d + %d", i, i)
		//接收消息
		rsp, err := session.Receive()
		checkErr(err)
		log.Printf("Receive: %d", rsp.(*AddRsp).C)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
