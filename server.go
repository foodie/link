package link

import "net"

//Server的基本类型
type Server struct {
	manager      *Manager     //session管理器
	listener     net.Listener //监听端口
	protocol     Protocol     //协议
	handler      Handler      //handler处理器
	sendChanSize int          //发送chan的size
}

//处理session的函数
type Handler interface {
	HandleSession(*Session) //处理session
}

//默认的处理函数
var _ Handler = HandlerFunc(nil)

//session处理器
type HandlerFunc func(*Session) //任意的处理session的函数

func (f HandlerFunc) HandleSession(session *Session) {
	f(session)
}

func NewServer(listener net.Listener, protocol Protocol, sendChanSize int, handler Handler) *Server {
	return &Server{
		manager:      NewManager(),
		listener:     listener,
		protocol:     protocol,
		handler:      handler,
		sendChanSize: sendChanSize,
	}
}

func (server *Server) Listener() net.Listener {
	return server.listener
}

func (server *Server) Serve() error {
	for {
		//获取net.Conn
		conn, err := Accept(server.listener)
		if err != nil {
			return err
		}
		//发起一个协成处理结果
		go func() {
			//使用protocol获取一个codec，主要用来读写数据
			codec, err := server.protocol.NewCodec(conn)
			if err != nil {
				conn.Close()
				return
			}
			//对当前连接新建一个session
			session := server.manager.NewSession(codec, server.sendChanSize)
			//处理连接
			server.handler.HandleSession(session)
		}()
	}
}

func (server *Server) GetSession(sessionID uint64) *Session {
	return server.manager.GetSession(sessionID)
}

func (server *Server) Stop() {
	server.listener.Close()
	server.manager.Dispose()
}
