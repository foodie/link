package link

import "sync"

const sessionMapNum = 32

//session的管理
type Manager struct {
	sessionMaps [sessionMapNum]sessionMap
	disposeOnce sync.Once
	disposeWait sync.WaitGroup
}

//session的基本信息
type sessionMap struct {
	sync.RWMutex
	sessions map[uint64]*Session
	disposed bool
}

//新建一个Manager，含有多个sessionMapNum个 sessionMap
func NewManager() *Manager {
	manager := &Manager{}
	for i := 0; i < len(manager.sessionMaps); i++ {
		manager.sessionMaps[i].sessions = make(map[uint64]*Session)
	}
	return manager
}

//只做一次
func (manager *Manager) Dispose() {
	manager.disposeOnce.Do(func() {
		for i := 0; i < sessionMapNum; i++ {
			//获取当前的sessionMap
			smap := &manager.sessionMaps[i]
			smap.Lock()
			smap.disposed = true //关闭
			//关闭当个sessionMap
			for _, session := range smap.sessions {
				session.Close()
			}
			smap.Unlock()
		}
		//等待线程组的结束
		manager.disposeWait.Wait()
	})
}

//新建一个session，把session放入manager
func (manager *Manager) NewSession(codec Codec, sendChanSize int) *Session {
	session := newSession(manager, codec, sendChanSize)
	manager.putSession(session)
	return session
}

//根据session_id获取一个session
func (manager *Manager) GetSession(sessionID uint64) *Session {
	smap := &manager.sessionMaps[sessionID%sessionMapNum]
	smap.RLock()
	defer smap.RUnlock()

	session, _ := smap.sessions[sessionID]
	return session
}

//根据session_id放入session
func (manager *Manager) putSession(session *Session) {
	smap := &manager.sessionMaps[session.id%sessionMapNum]

	smap.Lock()
	defer smap.Unlock()

	if smap.disposed {
		session.Close()
		return
	}

	smap.sessions[session.id] = session
	//增加一个Add+1
	manager.disposeWait.Add(1)
}

//删除一个session
func (manager *Manager) delSession(session *Session) {
	smap := &manager.sessionMaps[session.id%sessionMapNum]

	smap.Lock()
	defer smap.Unlock()

	delete(smap.sessions, session.id)
	//增加一个done-1
	manager.disposeWait.Done()
}
