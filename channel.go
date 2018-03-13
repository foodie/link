package link

import (
	"sync"
)

//定义一种类型
type KEY interface{}

type Channel struct {
	mutex    sync.RWMutex     //读写锁
	sessions map[KEY]*Session //一个map

	// channel state
	State interface{} //一个状态
}

//新建个channel
func NewChannel() *Channel {
	return &Channel{
		sessions: make(map[KEY]*Session),
	}
}

//chan的长度
func (channel *Channel) Len() int {
	channel.mutex.RLock()
	defer channel.mutex.RUnlock()
	return len(channel.sessions)
}

//对所有的session调用所有的回调函数
func (channel *Channel) Fetch(callback func(*Session)) {
	channel.mutex.RLock()
	defer channel.mutex.RUnlock()
	for _, session := range channel.sessions {
		callback(session)
	}
}

//获取一个session
func (channel *Channel) Get(key KEY) *Session {
	channel.mutex.RLock()
	defer channel.mutex.RUnlock()
	session, _ := channel.sessions[key]
	return session
}

//添加一个session
func (channel *Channel) Put(key KEY, session *Session) {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	if session, exists := channel.sessions[key]; exists {
		channel.remove(key, session)
	}
	//调用关闭的回调函数
	session.AddCloseCallback(channel, key, func() {
		channel.Remove(key)
	})
	//放入session
	channel.sessions[key] = session
}

//调用移除的回调函数，删除对应的session
func (channel *Channel) remove(key KEY, session *Session) {
	//移除和关闭时的回调
	session.RemoveCloseCallback(channel, key)
	delete(channel.sessions, key)
}

//删除对应的session
func (channel *Channel) Remove(key KEY) bool {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	session, exists := channel.sessions[key]
	if exists {
		channel.remove(key, session)
	}
	return exists
}

//获取一个连接，并且把它删除
func (channel *Channel) FetchAndRemove(callback func(*Session)) {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	for key, session := range channel.sessions {
		//移除同时关闭
		session.RemoveCloseCallback(channel, key)
		delete(channel.sessions, key)
		//调用回调
		callback(session)
	}
}

//关闭所有的连接
func (channel *Channel) Close() {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()
	for key, session := range channel.sessions {
		channel.remove(key, session)
	}
}
