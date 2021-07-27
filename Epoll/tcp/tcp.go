package tcpio

import (
	"golang.org/x/sys/unix"
	"log"
	"net"
	"reflect"
	"sync"
	"syscall"
)

type tcpepoll struct {
	epfd        int
	connections map[int]*TCPIO
	lock        *sync.RWMutex
}

func NewTCPEpoll() (*tcpepoll, error) {
	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	return &tcpepoll{
		epfd:        fd,
		lock:        &sync.RWMutex{},
		connections: make(map[int]*TCPIO),
	}, nil
}

func (e *tcpepoll) AddTCPIO(tcp *TCPIO) error {
	return e.Add(tcp.conn)
}

func (e *tcpepoll) Add(conn net.Conn) error {
	fd := netconnFD(conn)
	err := unix.EpollCtl(e.epfd, syscall.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Events: unix.POLLIN | unix.POLLHUP, Fd: int32(fd)})
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	tcpconn := &TCPIO{
		rmutex: &sync.Mutex{},
		wmutex: &sync.Mutex{},
		conn:   conn,
		fd:     fd,
	}

	e.connections[fd] = tcpconn
	if len(e.connections)%100 == 0 {
		log.Printf("Total number of connections: %v", len(e.connections))
	}
	return nil
}

func (e *tcpepoll) RemoveTCPIO(tcp *TCPIO) error {
	return e.Remove(tcp.conn)
}

func (e *tcpepoll) Remove(conn net.Conn) error {
	fd := netconnFD(conn)
	err := unix.EpollCtl(e.epfd, syscall.EPOLL_CTL_DEL, fd, nil)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.connections, fd)
	if len(e.connections)%100 == 0 {
		log.Printf("Total number of connections: %v", len(e.connections))
	}
	return nil
}

func (e *tcpepoll) Wait(timeoutMsec int) ([]net.Conn, error) {
	events := make([]unix.EpollEvent, 100)
	_, err := unix.EpollWait(e.epfd, events, timeoutMsec)
	if err != nil {
		return nil, err
	}
	e.lock.RLock()
	defer e.lock.RUnlock()
	var connections []net.Conn

	for _, ev := range events {
		tcp := e.connections[int(ev.Fd)]
		connections = append(connections, tcp.conn)
	}
	return connections, nil

}

func (e *tcpepoll) TCPWait(timeoutMsec int) ([]*TCPIO, error) {
	events := make([]unix.EpollEvent, 100)
	_, err := unix.EpollWait(e.epfd, events, timeoutMsec)
	if err != nil {
		return nil, err
	}
	e.lock.RLock()
	defer e.lock.RUnlock()
	var connections []*TCPIO

	for _, ev := range events {
		tcp := e.connections[int(ev.Fd)]
		connections = append(connections, tcp)
	}
	return connections, nil
}

func netconnFD(conn net.Conn) int {
	v := reflect.ValueOf(conn)
	netFD := reflect.Indirect(reflect.Indirect(v).FieldByName("fd"))
	pfd := reflect.Indirect(netFD.FieldByName("pfd"))
	fd := int(pfd.FieldByName("Sysfd").Int())
	return fd
}
