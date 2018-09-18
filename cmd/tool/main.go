package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/client"
	"github.com/taorenhai/ancestor/config"
	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util"
)

func toolInit() {
	log.Infof("tool init \n")
	file := flag.String("c", "", "config file")
	flag.Parse()

	cfg := getConfig(*file)

	etcds := cfg.GetString(config.SectionDefault, "EtcdHosts")
	nodes := cfg.GetString(config.SectionDefault, "Nodes")

	if err := toolClientInit(etcds, nodes); err != nil {
		log.Errorf("init data failure, %v", err)
		return
	}
	log.Infof("tool init success")
}

func getConfig(file string) *config.Config {
	if !util.IsFileExists(file) {
		panic("the config not exist, panic return")
	}

	log.Infof("config file path:%v", file)
	cfg, err := config.LoadConfig(file)
	if err != nil {
		panic(fmt.Sprintf("load config failure, err:%v", err))
	}

	return cfg
}

func toolClientInit(etcds string, nodes string) error {
	s, err := client.Open(etcds)
	if err != nil {
		log.Errorf("NewLocalClient err:%v", err)
		return err
	}
	defer s.Close()

	var nds []meta.NodeDescriptor
	var reps []meta.ReplicaDescriptor

	for i, host := range strings.Split(nodes, ";") {
		reps = append(reps, meta.ReplicaDescriptor{NodeID: meta.NodeID(i + 1), ReplicaID: meta.ReplicaID(i + 1)})
		nds = append(nds, meta.NodeDescriptor{NodeID: meta.NodeID(i + 1), Address: host})
	}

	if err = s.GetAdmin().SetNodes(nds); err != nil {
		return errors.Errorf("SetNodeList failure, err:%v", err)
	}
	log.Infof("tool init node descriptors success, nodes:%+v", nds)

	var rds []meta.RangeDescriptor
	for i := 1; i < 3; i++ {
		id, e := s.GetAdmin().GetNewID()
		if e != nil {
			return errors.Errorf("GetNewRangeID failure, err:%v", e)
		}

		rd := meta.RangeDescriptor{RangeID: id, StartKey: []byte{byte(i)}, EndKey: []byte{byte(i + 1)}, Replicas: reps}
		rds = append(rds, rd)
	}

	if err = s.GetAdmin().SetRangeDescriptors(rds); err != nil {
		return errors.Errorf("SetRangeDescriptors failure, err:%v", err)
	}
	log.Infof("tool init range descriptors success, rds:%+v", rds)

	return nil
}

func showOptions() {
	fmt.Println("----------  Usage  -------------")
	fmt.Println("tool -c config")
	fmt.Println("tool -s config")
	fmt.Println("--------------------------------")
}

func showRanges() {
	log.Infof("tool get range descriptors \n")
	file := flag.String("s", "", "config file")
	flag.Parse()

	cfg := getConfig(*file)
	etcds := cfg.GetString(config.SectionDefault, "EtcdHosts")

	s, err := client.Open(etcds)
	if err != nil {
		log.Errorf("tool new client err:%v", err)
		return
	}
	defer s.Close()

	rds, err := s.GetAdmin().GetRangeDescriptors()
	if err != nil {
		log.Errorf("tool get range descriptor err:%v", err)
		return
	}

	for _, rd := range rds {
		log.Infof("[%v] [%v, %v) replicas:%v", rd.RangeID,
			rd.StartKey, rd.EndKey, rd.Replicas)
	}

	log.Infof("tool get %v range descriptors", len(rds))
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	switch os.Args[1] {
	case "-c":
		toolInit()
	case "-s":
		showRanges()
	default:
		showOptions()
	}
}
