package client

import (
	"net"
	//"reflect"
	"sync"
	//	"sync/atomic"
	//"time"

	"github.com/juju/errors"
	"github.com/zssky/log"
	"golang.org/x/net/context"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/retry"
	"github.com/taorenhai/ancestor/util/stop"
)

// Sender rpc sender interface
type Sender interface {

	// send request wait response, or return error
	send(*meta.BatchRequest, *meta.BatchResponse) error

	// close rpc client
	stop()

	// return rpc server addr
	getAddr() string
}

// rpcSender rpc client
type rpcSender struct {
	retryOpts retry.Options
	stopper   *stop.Stopper
	addr      net.Addr
	client    *meta.NodeClient

	sync.Mutex
}

// newRPCSender returns a new instance of Sender.
func newRPCSender(addr net.Addr, retryOpts retry.Options) Sender {
	return &rpcSender{
		retryOpts: retryOpts,
		addr:      addr,
	}
}

func (rs *rpcSender) getRPCClient() (*meta.NodeClient, error) {
	rs.Lock()
	defer rs.Unlock()

	if rs.client != nil {
		return rs.client, nil
	}

	var err error

	rs.client, err = meta.NewNodeClient(rs.getAddr())

	return rs.client, err
}

/*
var (
	// ID for testing debug
	// TODO can remove after finishing test
	ID int64
)

// TODO can remove after finishing test
func getTxnID(req *meta.BatchRequest) string {
	if req.GetReq()[0].GetInner().Header().Txn != nil {
		return string(req.GetReq()[0].GetInner().Header().Txn.ID)
	}
	return ""
}
*/

// Send send rpc request, recv response
func (rs *rpcSender) send(req *meta.BatchRequest, resp *meta.BatchResponse) error {
	c, err := rs.getRPCClient()
	if err != nil {
		log.Errorf("getRPCClient error:%s", err.Error())
		return errors.Trace(err)
	}

	// TODO can remove after finishing test
	/*
		id := atomic.AddInt64(&ID, 1)
		reqStr := reflect.TypeOf(req.GetReq()[0].GetInner()).String()
		b := time.Now().UnixNano()
		log.Infof("%.8x being req:%s, batchSize:%d, host:%s, txn:%s", id, reqStr, len(req.GetReq()), rs.getAddr(), getTxnID(req))
	*/

	nodeResp, err := c.NodeMessage(context.TODO(), req)

	/*
		// TODO can remove after finishing test
		e := time.Now().UnixNano()
		log.Infof("%.8x end req:%s, used:%d", id, reqStr, time.Duration(e-b)/time.Millisecond)
	*/

	if err != nil {
		log.Errorf("NodeMessage error:%s", err.Error())
		rs.stop()
		return errors.Trace(meta.NewRPCConnectError(rs.getAddr()))
	}

	*resp = *nodeResp

	return nil
}

// close rpc client
func (rs *rpcSender) stop() {
	log.Warningf("close sender %s", rs.addr.String())
	rs.Lock()
	if rs.client != nil {
		rs.client.Close()
		rs.client = nil
	}
	rs.Unlock()
}

// getAddr remote server addr
func (rs *rpcSender) getAddr() string {
	return rs.addr.String()
}
