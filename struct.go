package test1

//api.go
type Protocol interface {
	NewCodec(rw io.ReadWriter) (Codec, error)
}

type ProtocolFunc func(rw io.ReadWriter) (Codec, error)

func (pf ProtocolFunc) NewCodec(rw io.ReadWriter) (Codec, error) {
	return pf(rw)
}

//读写数据
type Codec interface {
	Receive() (interface{}, error)
	Send(interface{}) error
	Close() error
}

type ClearSendChan interface {
	ClearSendChan(<-chan interface{})
}

//channel.go

type KEY interface{}

type Channel struct {
	mutex    sync.RWMutex
	sessions map[KEY]*Session

	// channel state
	State interface{}
}

//manager.go
type Manager struct {
	sessionMaps [sessionMapNum]sessionMap
	disposeOnce sync.Once
	disposeWait sync.WaitGroup
}

type sessionMap struct {
	sync.RWMutex
	sessions map[uint64]*Session
	disposed bool
}

//server.go
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

var _ Handler = HandlerFunc(nil)

type HandlerFunc func(*Session) //任意的处理session的函数

func (f HandlerFunc) HandleSession(session *Session) {
	f(session)
}

//session.go

type Session struct {
	id        uint64
	codec     Codec
	manager   *Manager
	sendChan  chan interface{}
	recvMutex sync.Mutex
	sendMutex sync.RWMutex

	closeFlag          int32
	closeChan          chan int
	closeMutex         sync.Mutex
	firstCloseCallback *closeCallback
	lastCloseCallback  *closeCallback

	State interface{}
}

//回调函数
type closeCallback struct {
	Handler interface{}
	Key     interface{}
	Func    func()
	Next    *closeCallback
}
