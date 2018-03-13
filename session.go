package link

import (
	"errors"
	"sync"
	"sync/atomic"
)

var SessionClosedError = errors.New("Session Closed")
var SessionBlockedError = errors.New("Session Blocked")

var globalSessionId uint64

//真正的session处理
type Session struct {
	id        uint64
	codec     Codec
	manager   *Manager
	sendChan  chan interface{} //发送数据
	recvMutex sync.Mutex
	sendMutex sync.RWMutex

	closeFlag          int32
	closeChan          chan int
	closeMutex         sync.Mutex
	firstCloseCallback *closeCallback
	lastCloseCallback  *closeCallback

	State interface{}
}

//新建一个客户端session
//获取一个无manage的session处理器
func NewSession(codec Codec, sendChanSize int) *Session {
	return newSession(nil, codec, sendChanSize)
}

//新建一个session
func newSession(manager *Manager, codec Codec, sendChanSize int) *Session {
	session := &Session{
		codec:     codec,
		manager:   manager,
		closeChan: make(chan int),
		id:        atomic.AddUint64(&globalSessionId, 1), //获取session的id
	}
	//获取接收的大小
	if sendChanSize > 0 {
		//新建一个sessionlop，如果是异步发送
		session.sendChan = make(chan interface{}, sendChanSize)
		go session.sendLoop()
	}
	return session
}

func (session *Session) ID() uint64 {
	return session.id
}

func (session *Session) IsClosed() bool {
	return atomic.LoadInt32(&session.closeFlag) == 1
}

func (session *Session) Close() error {
	if atomic.CompareAndSwapInt32(&session.closeFlag, 0, 1) {
		close(session.closeChan)

		if session.sendChan != nil {
			session.sendMutex.Lock()
			close(session.sendChan)
			if clear, ok := session.codec.(ClearSendChan); ok {
				clear.ClearSendChan(session.sendChan)
			}
			session.sendMutex.Unlock()
		}

		err := session.codec.Close()

		go func() {
			session.invokeCloseCallbacks()

			if session.manager != nil {
				session.manager.delSession(session)
			}
		}()
		return err
	}
	return SessionClosedError
}

func (session *Session) Codec() Codec {
	return session.codec
}

//接收数据
func (session *Session) Receive() (interface{}, error) {
	session.recvMutex.Lock()
	defer session.recvMutex.Unlock()

	//通过codec接收数据
	msg, err := session.codec.Receive()
	if err != nil {
		session.Close()
	}
	return msg, err
}

//不断的从chan里面读取数据，然后通过codec发送数据
func (session *Session) sendLoop() {
	defer session.Close()
	//1 不停的从session.sendChan里面获取数据
	//2 通过sendChan发送数据
	for {
		select {
		case msg, ok := <-session.sendChan:
			if !ok || session.codec.Send(msg) != nil {
				return
			}
		case <-session.closeChan:
			return
		}
	}
}

//发送数据
func (session *Session) Send(msg interface{}) error {

	//如果sendChan为空
	if session.sendChan == nil {
		//是否关闭
		if session.IsClosed() {
			return SessionClosedError
		}

		session.sendMutex.Lock()
		defer session.sendMutex.Unlock()
		//通过codec的send发送数据
		//直接接收和发送
		err := session.codec.Send(msg)
		if err != nil {
			session.Close()
		}
		return err
	}
	//读锁
	session.sendMutex.RLock()
	//session是否关闭，关闭返回错误
	if session.IsClosed() {
		session.sendMutex.RUnlock()
		return SessionClosedError
	}

	//发送数据到sendChan
	select {
	case session.sendChan <- msg:
		session.sendMutex.RUnlock()
		return nil
	default:
		session.sendMutex.RUnlock()
		session.Close()
		return SessionBlockedError
	}
}

type closeCallback struct {
	Handler interface{}
	Key     interface{}
	Func    func()
	Next    *closeCallback
}

func (session *Session) AddCloseCallback(handler, key interface{}, callback func()) {
	if session.IsClosed() {
		return
	}

	session.closeMutex.Lock()
	defer session.closeMutex.Unlock()

	newItem := &closeCallback{handler, key, callback, nil}

	if session.firstCloseCallback == nil {
		session.firstCloseCallback = newItem
	} else {
		session.lastCloseCallback.Next = newItem
	}
	session.lastCloseCallback = newItem
}

func (session *Session) RemoveCloseCallback(handler, key interface{}) {
	if session.IsClosed() {
		return
	}

	session.closeMutex.Lock()
	defer session.closeMutex.Unlock()

	var prev *closeCallback
	for callback := session.firstCloseCallback; callback != nil; prev, callback = callback, callback.Next {
		if callback.Handler == handler && callback.Key == key {
			if session.firstCloseCallback == callback {
				session.firstCloseCallback = callback.Next
			} else {
				prev.Next = callback.Next
			}
			if session.lastCloseCallback == callback {
				session.lastCloseCallback = prev
			}
			return
		}
	}
}

func (session *Session) invokeCloseCallbacks() {
	session.closeMutex.Lock()
	defer session.closeMutex.Unlock()

	for callback := session.firstCloseCallback; callback != nil; callback = callback.Next {
		callback.Func()
	}
}
