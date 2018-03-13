package link

import (
	"io"
	"net"
	"strings"
	"time"
)

//定义协议类型接口，通过一个读写类型，获取一个接口返回一个Codec
type Protocol interface {
	NewCodec(rw io.ReadWriter) (Codec, error)
}

//定义一个ProtocolFunc类型，调用它之后返回一个Codec
type ProtocolFunc func(rw io.ReadWriter) (Codec, error)

func (pf ProtocolFunc) NewCodec(rw io.ReadWriter) (Codec, error) {
	return pf(rw)
}

//接收，发送，关闭
type Codec interface {
	Receive() (interface{}, error)
	Send(interface{}) error
	Close() error
}

type ClearSendChan interface {
	ClearSendChan(<-chan interface{}) //接收一个读的chan
}

//新建一个server
func Listen(network, address string, protocol Protocol, sendChanSize int, handler Handler) (*Server, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return NewServer(listener, protocol, sendChanSize, handler), nil
}

//新建一个session
func Dial(network, address string, protocol Protocol, sendChanSize int) (*Session, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	codec, err := protocol.NewCodec(conn)
	if err != nil {
		return nil, err
	}
	return NewSession(codec, sendChanSize), nil
}

//新建一个超时的session
func DialTimeout(network, address string, timeout time.Duration, protocol Protocol, sendChanSize int) (*Session, error) {
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	codec, err := protocol.NewCodec(conn)
	if err != nil {
		return nil, err
	}
	return NewSession(codec, sendChanSize), nil
}

//获取连接
func Accept(listener net.Listener) (net.Conn, error) {
	var tempDelay time.Duration
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			if strings.Contains(err.Error(), "use of closed network connection") {
				return nil, io.EOF
			}
			return nil, err
		}
		return conn, nil
	}
}
