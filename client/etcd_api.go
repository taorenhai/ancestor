package client

import (
	"encoding/json"
	"net"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"
	"github.com/zssky/log"
	"golang.org/x/net/context"

	"github.com/taorenhai/ancestor/util"
)

const (
	masterHostKey  = util.ETCDRootPath + "/MasterHost"
	connectTimeout = 5 * time.Second
)

// etcdAPI
type etcdAPI struct {
	etcdClient *clientv3.Client
}

var (
	errMasterNotFound = errors.New("etcd master host not found")
)

type masterHost struct {
	Master string
}

func newEtcdAPI(addrs []string) (*etcdAPI, error) {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   addrs,
		DialTimeout: connectTimeout,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &etcdAPI{etcdClient: c}, nil
}

// close the sender
func (m *etcdAPI) stop() {
	m.etcdClient.Close()
}

func (m *etcdAPI) getMasterAddr() (net.Addr, error) {
	kv := clientv3.NewKV(m.etcdClient)
	ctx, cancel := context.WithTimeout(context.Background(), defaultRetryOptions.MaxBackoff)
	resp, err := kv.Get(ctx, masterHostKey)
	cancel()
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(resp.Kvs) != 1 {
		return nil, errors.Trace(errMasterNotFound)
	}

	var mh masterHost

	if err = json.Unmarshal(resp.Kvs[0].Value, &mh); err != nil {
		return nil, errors.Annotatef(err, "Unmarshal etcd response error")
	}

	addr, err := net.ResolveTCPAddr("tcp", mh.Master)
	if err != nil {
		return nil, errors.Trace(err)
	}

	log.Debugf("find host:%s", mh.Master)

	return addr, nil
}
