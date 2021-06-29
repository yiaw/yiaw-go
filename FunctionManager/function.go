package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	NONE_RECVER_MSGID = 10000001
	SEND_RECVER_MSGID = 10000002
)

type Message struct {
	MsgId int
	Data  interface{}
}

type Manager struct {
	name   string
	wg     *sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	mutex  *sync.RWMutex

	timer     *time.Ticker
	checkTime time.Time

	sendStopFlag int
	channel      chan Message

	recvHandler    func(m *Manager, msgid int, data interface{}, arg interface{})
	rcvarg         interface{}
	sendNoneRecver func(m *Manager, msgid int, data interface{}, arg interface{})
	snrarg         interface{}
	timeOutHook    func(m *Manager, arg interface{})
	toarg          interface{}
	fullHook       func(m *Manager, channel chan Message, msg Message, arg interface{})
	farg           interface{}
}

var fml map[string]*Manager

func GetManager(name string) *Manager {
	m, ok := fml[name]
	if !ok {
		return nil
	}
	return m
}

func NewManagerInit(name string, channelMax int, timeoutMsec int) (*Manager, error) {
	_, ok := fml[name]

	if ok {
		return nil, fmt.Errorf("already exsits Manager %s\n", name)
	}

	timeout := time.Duration(timeoutMsec) * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	man := &Manager{
		name:   name,
		wg:     &sync.WaitGroup{},
		ctx:    ctx,
		cancel: cancel,
		mutex:  &sync.RWMutex{},

		timer:     time.NewTicker(timeout),
		checkTime: time.Now(),

		channel: make(chan Message, channelMax),
	}
	if fml == nil {
		fml = make(map[string]*Manager)
	}

	fmt.Println("Init", cap(man.channel), len(man.channel))
	fml[name] = man
	return man, nil
}

func NewManager() *Manager {
	man := &Manager{}
	return man
}

func (m *Manager) GetInstance(name string) error {
	man, ok := fml[name]
	if !ok {
		return errors.New("not found Manager instance")
	}
	m = man
	return nil
}

func (m *Manager) SetRecvHandler(hdrl func(m *Manager, msgId int, data interface{}, arg interface{}),
	uarg interface{}) {
	m.recvHandler = hdrl
	m.rcvarg = uarg
}

func (m *Manager) SetSendNoneRecvHandler(hdrl func(m *Manager, msgId int, data interface{},
	arg interface{}), uarg interface{}) {
	m.sendNoneRecver = hdrl
	m.snrarg = uarg
}

func (m *Manager) SetTimeOutHook(hdrl func(m *Manager, arg interface{}), uarg interface{}) {
	m.timeOutHook = hdrl
	m.toarg = uarg
}

func (m *Manager) SetChannelFullHook(
	hdrl func(m *Manager, channel chan Message, msg Message, arg interface{}), uarg interface{}) {
	m.fullHook = hdrl
	m.farg = uarg
}

func (m *Manager) Start() {
	go func(m *Manager) {
		for {
			select {
			case msg := <-m.channel:
				if m.recvHandler != nil {
					m.recvHandler(m, msg.MsgId, msg.Data, m.rcvarg)
				}
			case <-m.ctx.Done():
				// Destory
				return
			case <-m.timer.C:
				if m.timeOutHook != nil {
					m.timeOutHook(m, m.toarg)
				}
			}
		}

	}(m)
}

func (m *Manager) SetNoneRecver() {
	m.setSendFlag(1)
}

func (m *Manager) SetSendRecver() {
	m.setSendFlag(0)
}

func (m *Manager) SendMessage(msgid int, data interface{}) {
	message := Message{
		MsgId: msgid,
		Data:  data,
	}

	if cap(m.channel) == len(m.channel) {
		if m.fullHook != nil {
			m.fullHook(m, m.channel, message, m.farg)
		}
	} else if m.getSendFlag() > 0 {
		if m.sendNoneRecver != nil {
			m.sendNoneRecver(m, msgid, data, m.snrarg)
		}
	} else {
		m.channel <- message
	}
}

func (m *Manager) getSendFlag() int {
	m.mutex.RLock()
	opt := m.sendStopFlag
	m.mutex.RUnlock()
	return opt
}

func (m *Manager) setSendFlag(opt int) {
	m.mutex.Lock()
	m.sendStopFlag = opt
	m.mutex.Unlock()
}

/* One Destroy */
func (m *Manager) Destroy() {
	m.cancel()
	m.wg.Wait()
	delete(fml, m.name)
}

/* All Destroy */
func ManagerDestroy() {
	for n, m := range fml {
		m.cancel()
		m.wg.Wait()
		delete(fml, n)
	}
}
