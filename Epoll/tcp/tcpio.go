package tcpio

import (
	"encoding/json"
	"errors"
	_ "fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"telcoware.com/lib/log"
	"time"
)

var TIMEOUT = errors.New("TIMEOUT")
var DISCONNECT = io.EOF
var EOF = io.EOF

type TCPIO struct {
	rmutex *sync.Mutex
	wmutex *sync.Mutex
	conn   net.Conn
	fd     int
	once   sync.Once
}

func NewTCPIO() *TCPIO {
	return &TCPIO{
		rmutex: &sync.Mutex{},
		wmutex: &sync.Mutex{},
	}
}

func (t *TCPIO) RowFd() int {
	t.once.Do(func() {
		t.fd = netconnFD(t.conn)
	})
	return t.fd
}

func (t *TCPIO) readHeader(expireTime *time.Time) (int, error) {
	var dat map[string]interface{}

	buf := make([]byte, 1024)
	readSize := 0
	for {
		if expireTime != nil {
			curTime := time.Now()
			if expireTime.Sub(curTime).Milliseconds() < 0 {
				log.DBG("%s:: Read Timeout", log.FUNC())
				return -1, TIMEOUT
			}
			t.conn.SetDeadline(time.Now().Add(10 * time.Millisecond))
		}

		nbytes, err := t.conn.Read(buf[readSize : readSize+1])
		if err != nil || nbytes <= 0 {
			if err, ok := err.(*net.OpError); ok && err.Timeout() {
				return -1, TIMEOUT
			}

			if err == io.EOF {
				log.ERR("%s:: Connection is Closed from client; %v\n", log.FUNC(), t.conn.RemoteAddr().String())
				return -1, DISCONNECT
			}

			log.ERR("%s:: Read() failed. err=%s", log.FUNC(), err)
			return -1, err
		}

		readSize += nbytes

		if readSize > 1 && buf[readSize-2] == '\r' && buf[readSize-1] == '\n' {
			break
		}
	}

	err := json.Unmarshal(buf[:readSize-2], &dat)
	if err != nil {
		log.ERR("%s:: json.Unmarshal() failed. err=%s", log.FUNC(), err)
		return -1, err
	}
	return int(dat["msgSize"].(float64)), nil
}

func (t *TCPIO) readBody(msgSize int, expireTime *time.Time) (int, []byte, error) {
	//var body []byte
	body := make([]byte, msgSize, msgSize)
	readSize := 0
	remainSize := msgSize
	for {
		if remainSize == 0 {
			break
		}

		if expireTime != nil {
			curTime := time.Now()
			if expireTime.Sub(curTime).Milliseconds() < 0 {
				log.DBG("%s:: Read Timeout", log.FUNC())
				return -1, nil, TIMEOUT
			}
			t.conn.SetDeadline(time.Now().Add(10 * time.Millisecond))
		}

		nbytes, err := t.conn.Read(body[readSize:msgSize])
		if err != nil || nbytes <= 0 {
			if io.EOF == err {
				log.ERR("%s:: Connection is Closed from client; %v\n", t.conn.RemoteAddr().String())
				return -1, nil, DISCONNECT
			}
			log.ERR("%s:: Read() failed. err=%s", log.FUNC(), err)
			return -1, nil, err
		}

		readSize += nbytes
		remainSize -= nbytes
	}
	return readSize, body, nil
}

func (t *TCPIO) ReadMessage(timeoutMsec int) (int, []byte, error) {
	var timeout *time.Time = nil
	if t == nil {
		return -1, nil, errors.New("invalid memory")
	}

	if timeoutMsec > 0 {
		to := time.Now().Add(time.Duration(timeoutMsec) * time.Millisecond)
		timeout = &to
	}

	t.rmutex.Lock()
	defer t.rmutex.Unlock()
	size, err := t.readHeader(timeout)
	if err != nil {
		return size, nil, err
	}

	if size <= 0 {
		return size, nil, errors.New("Body Message Size Zero")
	}

	n, body, err := t.readBody(size, timeout)

	return n, body, err
}

func (t *TCPIO) writeHeader(msgSize int, expireTime *time.Time) (int, error) {
	var sendData int
	var header string
	var headerSize int

	header_len := len("{\"msgSize\":" + strconv.Itoa(msgSize) + "}\r\n")
	header = "{\"msgSize\":" + strconv.Itoa(msgSize) + "}"
	for i := 0; i < 24-header_len; i++ {
		header += " "
	}
	header += "\r\n"

	headerSize = len(header)
	sendData = 0

	for {
		if sendData == headerSize {
			break
		}

		if expireTime != nil {
			curTime := time.Now()
			if expireTime.Sub(curTime).Milliseconds() < 0 {
				log.DBG("%s:: Write Timeout", log.FUNC())
				return -1, TIMEOUT
			}
			t.conn.SetDeadline(time.Now().Add(10 * time.Millisecond))
		}

		nbytes, err := t.conn.Write([]byte(header[sendData:headerSize]))
		if err != nil || nbytes <= 0 {
			log.ERR("%s:: Write() failed. err=%s", log.FUNC(), err)
			return -1, err
		}
		sendData += nbytes
	}
	return sendData, nil
}

func (t *TCPIO) writeBody(sendMsg []byte, msgSize int, expireTime *time.Time) (int, error) {
	sendData := 0
	for {
		if sendData == msgSize {
			break
		}
		if expireTime != nil {
			curTime := time.Now()
			if expireTime.Sub(curTime).Milliseconds() < 0 {
				log.DBG("%s:: Write Timeout", log.FUNC())
				return -1, TIMEOUT
			}
			t.conn.SetDeadline(time.Now().Add(10 * time.Millisecond))
		}

		nbytes, err := t.conn.Write(sendMsg[sendData:msgSize])
		if err != nil || nbytes <= 0 {
			log.ERR("%s:: Write() failed. err=%s", log.FUNC(), err)
			return -1, err
		}
		sendData += nbytes
	}
	return sendData, nil
}

func (t *TCPIO) WriteMessage(sendMsg []byte, timeoutMsec int) (int, error) {
	var timeout *time.Time = nil
	if t == nil {
		return -1, errors.New("invalid memory")
	}

	if timeoutMsec > 0 {
		to := time.Now().Add(time.Duration(timeoutMsec) * time.Millisecond)
		timeout = &to
	}

	msgSize := int(len(sendMsg))

	t.wmutex.Lock()
	defer t.wmutex.Unlock()

	ret, err := t.writeHeader(msgSize, timeout)
	if err != nil {
		return ret, err
	}

	return t.writeBody(sendMsg, msgSize, timeout)
}

func (t *TCPIO) Close() {
	t.conn.Close()
}
