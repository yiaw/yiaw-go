package main

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	Main    *log.Logger
	Recv    *log.Logger
	Timeout *log.Logger
	Full    *log.Logger
}

var l Logger

func recvhdlr(m *Manager, msgid int, data interface{}, uarg interface{}) {
	Prefix := uarg.(string)
	if msgid == 2 {
		l.Recv.Printf("%s:: MSGID[%d], DATA[%v] Setting Send None Recver \n", Prefix, msgid, data)
	} else {
		l.Recv.Printf("%s:: MSGID[%d], DATA[%v]\n", Prefix, msgid, data)
	}
}

func tohdlr(m *Manager, uarg interface{}) {
	Prefix := uarg.(string)
	l.Timeout.Printf("%s:: [TIMEOUT]\n", Prefix)
}

func fullhdlr(m *Manager, channel chan Message, msg Message, uarg interface{}) {
	Prefix := uarg.(string)
	l.Full.Printf("%s:: Channel Full.. cap=%d, len=%d\n", Prefix, cap(channel), len(channel))

	for {
		if len(channel) == 0 {
			break
		}

		inner := <-channel
		l.Full.Printf("Channel Out,,, [%d][%s]\n", inner.MsgId, inner.Data)
	}
}

func nonsend(m *Manager, msgid int, data interface{}, uarg interface{}) {
	Prefix := uarg.(string)
	l.Main.Printf("%s:: MSGID[%d], DATA[%v]\n", Prefix, msgid, data)
	if msgid == 3 {
		l.Recv.Printf("%s:: MSGID[%d], DATA[%v] Setting Send Recver \n", Prefix, msgid, data)
	}
}

func LogInit() {
	l.Main = log.New(os.Stdout, "[Main]", log.LstdFlags)
	l.Recv = log.New(os.Stdout, "[Recv]", log.LstdFlags)
	l.Timeout = log.New(os.Stdout, "[Timeout]", log.LstdFlags)
	l.Full = log.New(os.Stdout, "[Full]", log.LstdFlags)
}
func main() {
	LogInit()
	m, err := NewManagerInit("Elasticsearch", 5, 5000)
	if err != nil {
		l.Main.Println("Init Fail..", err.Error())
		return
	}

	m.SetRecvHandler(recvhdlr, "RecvMessage")
	m.SetSendNoneRecvHandler(nonsend, "Send None Recver")
	m.SetTimeOutHook(tohdlr, "TimeOutMessage")
	m.SetChannelFullHook(fullhdlr, "Message")
	m.Start()

	var num int
	for {
		fmt.Scanf("%d", &num)
		fmt.Println(">> ", num)
		if num == 0 {
			// quit
			break
		} else if num == 1 {
			// One Message
			l.Main.Println("call SendMessage")
			m.SendMessage(num, "One Message")
		} else if num == 2 {
			// set None Send Flags Setting
			m.SetNoneRecver()
		} else if num == 3 {
			// set Send Flags Setting
			m.SetSendRecver()
		} else if num == 4 {
			// Bulk Message
			for i := 0; i < 1000; i++ {
				m.SendMessage(i, "BulkMessage")
			}
		}

	}
	/*
			time.Sleep(2 * time.Second)
			for i := 0; i < 100; i++ {
				m.SendMessage(i, "Test")
			}

		var t string
		fmt.Scanf("%s", &t)
	*/

}
