package main

import (
	"environment/cfgargs"
	"environment/logger"
	"fmt"
	"network/register"
	"network/server"
	"network/session"
)

var (
	BuildVersion = ""
)

func main() {
	srvCfg, err := cfgargs.InitSrvConfig(BuildVersion, func() {
		// user flag binding code
	})
	if nil != err {
		fmt.Println(err)
		return
	}
	logger.Info("start init log")
	logger.InitLogger(srvCfg.Log.Path, srvCfg.Log.Console, srvCfg.Log.Level)
	logger.Info("end init log")

	register.RegisterSession(MySession{})
	srv := server.NewTCPServer()
	srv.Run(srvCfg.Info.Addr, 0, 0, 0, 0, 0)
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
	sb := m.BaseSession.ReadBuffer.FrontSlidingBuffer()
	logger.Debugf("MySession.OnRead %v", sb.Read(0))
	m.BaseSession.ReadBuffer.RemoveFrontSlidingBuffer()

	m.SendString("Meizizi")
}
