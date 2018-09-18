package meta

import (
	"time"

	"github.com/zssky/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	networkTimeout = 5 * time.Second
)

// NodeClient node rpc client.
type NodeClient struct {
	host string
	NodeMessageClient
	*grpc.ClientConn
}

// NewNodeClient mainly create client for sending message to node server.
func NewNodeClient(host string) (*NodeClient, error) {
	conn, err := grpcDial(host)
	if err != nil {
		return nil, err
	}

	c := &NodeClient{host: host, ClientConn: conn, NodeMessageClient: NewNodeMessageClient(conn)}
	go c.run()
	return c, nil
}

func (c *NodeClient) run() {
	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		<-t.C
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err := c.NodeMessageClient.HeartbeatMessage(ctx, &HeartbeatRequest{})
		cancel()
		if err != nil {
			log.Infof("connect close %s %s", c.host, err.Error())
			c.Close()
			return
		}
	}
}

// Close close rpc connect.
func (c *NodeClient) Close() {
	c.ClientConn.Close()
}

// NewRaftClient mainly create client for sending raft message to raft group peer.
func NewRaftClient(host string) (RaftMessageClient, *grpc.ClientConn, error) {
	conn, err := grpcDial(host)
	if err != nil {
		return nil, nil, err
	}

	client := NewRaftMessageClient(conn)

	return client, conn, nil
}

func grpcDial(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOpt := grpc.WithInsecure()
	dialOpts := make([]grpc.DialOption, 0, 2+len(opts))
	dialOpts = append(dialOpts, dialOpt, grpc.WithTimeout(networkTimeout))
	dialOpts = append(dialOpts, opts...)

	conn, err := grpc.Dial(target, dialOpts...)
	if err != nil {
		log.Errorf("target:%v grpc dial error:%v", target, err)
		return nil, err
	}

	return conn, err
}
