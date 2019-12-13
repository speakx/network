package main

import (
	"environment/cfgargs"
	"environment/dump"
	"environment/logger"
	"flag"
	"fmt"
	"network/bufpool"
	"network/client"
	"network/register"
	"network/server"
	"network/session"
	"sync"
	"time"
)

var (
	BuildVersion = ""
	ClientFlg    *bool
)

func main() {
	srvCfg, err := cfgargs.InitSrvConfig(BuildVersion, func() {
		// user flag binding code
		ClientFlg = flag.Bool("client", false, "(default false)")
	})
	if nil != err {
		fmt.Println(err)
		return
	}

	if false == *ClientFlg {
		logger.Info("start init log")
		logger.InitLogger(srvCfg.Log.Path+".server", srvCfg.Log.Console, srvCfg.Log.Level)
		logger.Info("end init log")

		register.RegisterSession(MySession{})
		srv := server.NewTCPServer()
		srv.Run(srvCfg.Info.Addr, 0, 0, 0, 0, 0)
	} else {
		logger.Info("start init log")
		logger.InitLogger(srvCfg.Log.Path+".client", srvCfg.Log.Console, srvCfg.Log.Level)
		logger.Info("end init log")

		dump.InitDump(true, srvCfg.Dump.Interval, srvCfg.Dump.Addr,
			func(packetRecv, packetSend, packetRecvHandleRate, packetSendHandleRate int64) {
				logger.Info("dump rate recv:", packetRecvHandleRate, " send:", packetSendHandleRate)
			})

		for i := 0; i < 100; i++ {
			go func() {
				c := &MyClient{}
				client.TCPClientConnect("10.211.55.28:10001", time.Second, c)
				logger.Info("client.connect...", c)
				for {
					dump.NetEventSendIncr(0)
					c.WriteString("meizizi")
					dump.NetEventSendDecr(0)
				}
			}()
		}

		w := sync.WaitGroup{}
		w.Add(1)
		w.Wait()
	}
}

type MySession struct {
	session.BaseSession
}

func (m *MySession) OnOpen() {
	logger.Debugf("MySession.OnOpen")
	m.WriteString("Hello Client - from server")
}

func (m *MySession) OnClose() {
	logger.Debugf("MySession.OnClose")
}

func (m *MySession) OnRead() {
	sb := m.BaseSession.ReadBuffer.FrontSlidingBuffer()
	if nil != sb {
		logger.Debugf("MySession.OnRead %v", sb.GetWrited(0))
		m.BaseSession.ReadBuffer.RemoveFrontSlidingBuffer()
	}

	m.WriteString("echo...")
}

type MyClient struct {
	client.BaseClient
}

func (m *MyClient) OnRead(sb *bufpool.SlidingBuffer) {
	logger.Debugf("MyClient.OnRead %v", sb.GetWrited(0))
}
