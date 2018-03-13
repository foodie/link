package link

import "net"

//服务的struct
type Server struct {
	manager  *Manager     //一个manager
	listener net.Listener //一个listener

	protocol     Protocol //接收读写的协议接口
	handler      Handler  //处理session的接口
	sendChanSize int      //发送chan的大小
}

//处理session
type Handler interface {
	HandleSession(*Session)
}

//定义一个空函数
var _ Handler = HandlerFunc(nil)

//定义处理session的函数类型
type HandlerFunc func(*Session)

func (f HandlerFunc) HandleSession(session *Session) {
	f(session)
}

//新建一个server
func NewServer(listener net.Listener, protocol Protocol, sendChanSize int, handler Handler) *Server {
	return &Server{
		manager:      NewManager(),
		listener:     listener,
		protocol:     protocol,
		handler:      handler,
		sendChanSize: sendChanSize,
	}
}

//返回server
func (server *Server) Listener() net.Listener {
	return server.listener
}

//监听服务
func (server *Server) Serve() error {
	for {
		conn, err := Accept(server.listener)
		if err != nil {
			return err
		}

		go func() {
			//返回一个Codec接口类型
			codec, err := server.protocol.NewCodec(conn)
			if err != nil {
				conn.Close()
				return
			}
			//新建一个session
			session := server.manager.NewSession(codec, server.sendChanSize)
			//处理session
			server.handler.HandleSession(session)
		}()
	}
}

//获取session
func (server *Server) GetSession(sessionID uint64) *Session {
	return server.manager.GetSession(sessionID)
}

//停止服务
func (server *Server) Stop() {
	server.listener.Close()
	server.manager.Dispose()
}
