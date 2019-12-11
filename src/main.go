package main

import (
	"environment/cfgargs"
	"environment/logger"
	"fmt"
	"network/bufpool"
	"network/reactor"
	"network/server"
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

	reactor.InitNetworkHandler(&NetworkHandlerImp{})
	srv := server.NewTCPServer(0, 0, 0, 0)
	srv.Run(srvCfg.Info.Addr)
}

type NetworkHandlerImp struct {
}

func (n *NetworkHandlerImp) OnRead(session reactor.Session, sb *bufpool.SlidingBuffer) {
	logger.Debug("onread sid:", session.Sid(), " data:", sb.Read(0))
	session.SendString("Hehe")
}

func (n *NetworkHandlerImp) OnReadFinish(session reactor.Session) {

}
