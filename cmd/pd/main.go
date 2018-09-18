package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/taorenhai/ancestor/pd"
	"github.com/zssky/log"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	filePath := flag.String("c", "", "config file")
	debug := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	cfg, err := pd.NewConfig(*filePath, *debug)
	if err != nil {
		log.Errorf("load pd server config failed, err %s\n", err)
		return
	}

	server, err := pd.NewServer(cfg, *debug)
	if err != nil {
		log.Errorf("create pd server err %s\n", err)
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		sig := <-sc
		log.Infof("Got signal [%d] to exit.", sig)
		server.Stop()
		os.Exit(0)
	}()

	server.Start()
}
