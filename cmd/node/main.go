package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/config"
	"github.com/taorenhai/ancestor/storage"
	"github.com/taorenhai/ancestor/util"
)

func serverInit() {

	log.Infof("server init")

	configFile := flag.String("c", "", "config file")
	debug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	if !util.IsFileExists(*configFile) {
		panic("the config not exist, panic return")
	}

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		panic(fmt.Sprintf("load config failure, err:%v", err))
	}

	// set log info
	log.SetLevelByString(cfg.GetString("Log", "Level"))
	if !*debug {
		log.SetOutputByName(cfg.GetString("Log", "Path"))
		log.SetRotateByDay()
	}

	nodeServer, err := storage.NewNodeServer(cfg)
	if err != nil {
		log.Errorf(errors.ErrorStack(err))
		panic("create node server failure")
	}

	err = nodeServer.Start()
	if err != nil {
		log.Errorf(errors.ErrorStack(err))
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	go func() {
		fmt.Printf("%s", http.ListenAndServe(":9999", nil))
	}()
	serverInit()
}
