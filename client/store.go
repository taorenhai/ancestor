package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
	"github.com/taorenhai/ancestor/util/retry"
)

// defaultRetryOptions sets the retry options for handling retryable errors and
// connection I/O errors.
var defaultRetryOptions = retry.Options{
	InitialBackoff: 250 * time.Millisecond,
	MaxBackoff:     time.Second,
	MaxRetries:     50,
}

// storage implement Storage interface.
type storage struct {
	hosts []string
	*base
	*factory
	*heartbeat
	*admin
}

var (
	_ Storage = (*storage)(nil)

	// ErrInvalidState client need Open first.
	ErrInvalidState = errors.New("invalid client state")

	// ErrInvalidKey key is nil.
	ErrInvalidKey = errors.New("invalid key")

	// ErrInvalidValue value is nil.
	ErrInvalidValue = errors.New("value is nil")

	useNewVersion = (*meta.Timestamp)(nil)
)

// Open create ancestor client with given path, format shuld be: ip:port;ip:port;ip:port
func Open(path string) (Storage, error) {
	hosts := strings.Split(path, ";")

	b := newBase(hosts)

	a := newAdmin(b)

	if err := b.start(a); err != nil {
		log.Errorf("newBaseClient error:%s, host:%v", errors.ErrorStack(err), hosts)
		return nil, errors.Trace(err)
	}

	f := newFactory()
	f.start()

	h := newHeartbeat(b, f, a)
	h.start()

	return &storage{
		hosts:     hosts,
		base:      b,
		factory:   f,
		heartbeat: h,
		admin:     a,
	}, nil
}

//Begin implements engine begin, return new transaction
func (s *storage) Begin(priority ...int) (Transaction, error) {
	if s.base == nil {
		log.Errorf("need open first")
		return nil, ErrInvalidState
	}

	p := 0
	if len(priority) == 1 {
		p = priority[0]
	}

	txn, err := newTransaction(s.base, s.admin, s.factory, s.heartbeat, useNewVersion, p)
	if err != nil {
		log.Errorf("BeginTransaction error:%s", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}
	return txn, nil
}

// Close implements engine close all connect
func (s *storage) Close() error {
	if s.base == nil {
		log.Errorf("need open first")
		return ErrInvalidState
	}
	log.Info("close all connect")
	s.base.stop()
	s.factory.stop()
	s.heartbeat.stop()
	s.admin.stop()
	s.base = nil
	return nil
}

// CurrentVersion implements engine CurrentVersion, return current timestamp
func (s *storage) CurrentVersion() (meta.Timestamp, error) {
	if s.base == nil {
		log.Errorf("need open first")
		return meta.Timestamp{}, ErrInvalidState
	}
	return s.GetTimestamp()
}

// GetSnapshot implements engine GetSnapshot by timestamp
func (s *storage) GetSnapshot(ver meta.Timestamp) (Snapshot, error) {
	if s.base == nil {
		log.Errorf("need open first")
		return nil, ErrInvalidState
	}

	txn, err := newTransaction(s.base, s.admin, s.factory, s.heartbeat, &ver, 0)
	if err != nil {
		log.Errorf("beginTransaction error:%s", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	log.Debugf("new snapshot version:%s", ver.String())
	return txn.snapshot, nil
}

// UUID implements engine UUID return current conenct dsn
func (s *storage) UUID() string {
	return fmt.Sprintf("%v", s.hosts)
}

// GetAdmin return Admin interface.
func (s *storage) GetAdmin() Admin {
	return s.admin
}
