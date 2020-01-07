package main

import (
	"environment/cfgargs"
	"environment/dump"
	"environment/logger"
	"flag"
	"fmt"
	"network/client"
	"network/netpacket"
	"network/reactor"
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

	dump.InitDump(true, srvCfg.Dump.Interval, srvCfg.Dump.Addr,
		func(packetRecv, packetSend, packetRecvHandleRate, packetSendHandleRate int64) {
			fmt.Printf("dump rate recv:%v send:%v \n", packetRecvHandleRate, packetSendHandleRate)
		})

	if false == *ClientFlg {
		logger.Info("start init log")
		logger.InitLogger(srvCfg.Log.Path+".server", srvCfg.Log.Console, srvCfg.Log.Level)
		logger.Info("end init log")

		register.RegisterFactory(MySession{}, &MyTranslator{})
		srv := server.NewTCPServer()
		srv.Run(srvCfg.Info.Addr, 0, 0, 0, 0, 0)
	} else {
		logger.Info("start init log")
		logger.InitLogger(srvCfg.Log.Path+".client", srvCfg.Log.Console, srvCfg.Log.Level)
		logger.Info("end init log")

		register.RegisterFactory(MySession{}, &MyTranslator{})
		num := 10
		for i := 0; i < num; i++ {
			go func(idx int) {
				c := &MyClient{}
				client.TCPClientConnect("10.211.55.28:3389", time.Second, c)
				logger.Info("client.connect...", c)
				for seq := 0; ; seq++ {
					// <-time.After(time.Second)
					if nil == c.WriteByte([]byte(fmt.Sprintf("[%v][%v]meizizi", idx, seq))) {
					}
				}
				w := sync.WaitGroup{}
				w.Add(1)
				w.Wait()
			}(i)
		}

		w := sync.WaitGroup{}
		w.Add(1)
		w.Wait()
	}
}

type MyTranslator struct {
	buf []byte
	sync.Mutex
}

func (m *MyTranslator) Pack(handler reactor.TranslaterBuffer, data []byte) []byte {
	m.Lock()
	if m.buf == nil {
		m.buf = make([]byte, len(data)+4)
		netpacket.Uint32ToBytes(uint32(len(data)), m.buf)
		copy(m.buf[4:], data)
	}
	m.Unlock()
	return m.buf
	// copy(buf, data)
	// return len(data)
}

func (m *MyTranslator) UnPack(handler reactor.TranslaterBuffer, buf []byte) (int, []byte) {
	n := 0
	tag, tagSize := handler.GetReadTag()
	if 0 == tag {
		if len(buf) < 4 {
			return 0, nil
		}
		n += 4
		tag = 1
		tagSize = int(netpacket.BytesToUint32(buf))
		handler.SetReadTag(tag, tagSize)
		buf = buf[4:]
	}

	if 1 == tag && len(buf) >= tagSize {
		handler.SetReadTag(0, 4)
		return n + tagSize, buf[:tagSize]
	}

	return n, nil
	// return len(buf), buf
}

func (m *MyTranslator) GetHeadTag(handler reactor.TranslaterBuffer) (int, int) {
	return 0, 4
}

type MySession struct {
	session.BaseSession
}

func (m *MySession) OnOpen() {
	logger.Debugf("MySession.OnOpen")
}

func (m *MySession) OnClose() {
	logger.Debugf("MySession.OnClose")
}

func (m *MySession) OnRead() {
	for {
		sb := m.BaseSession.PopOnReadPacket()
		if nil != sb {
			dump.NetEventRecvIncr(0)
			// logger.Debugf("MySession.OnRead %v", string(sb.GetWrited(0)))
			m.WriteByte([]byte(fmt.Sprintf("%v echo...", string(sb.GetWrited(0)))))
			// m.WriteByte([]byte("echo..."))
			sb.ReleaseSlidingBuffer()
			dump.NetEventRecvDecr(0)
		} else {
			break
		}
	}
	m.BaseSession.OnRead()
}

type MyClient struct {
	client.BaseClient
}

func (m *MyClient) OnRead(data []byte) {
	dump.NetEventRecvIncr(0)
	logger.Debugf("MyClient.OnRead %v", string(data))
	dump.NetEventRecvDecr(0)
}
