package pd

import (
	"path"
	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"

	"github.com/taorenhai/ancestor/util"
	"github.com/taorenhai/ancestor/util/retry"
)

type idAllocator struct {
	etcd master
	key  string
}

func newIDAllocator(etcd master) *idAllocator {
	return &idAllocator{etcd: etcd, key: path.Join(util.ETCDRootPath, "NewID")}
}

func (i *idAllocator) newID() (int64, error) {
	var (
		cmp clientv3.Cmp
		id  int64
	)

	for r := retry.Start(retry.Options{}); r.Next(); {
		kvs, err := i.etcd.getValues(i.key)
		if err != nil {
			return 0, errors.Trace(err)
		}

		if kvs == nil {
			// create the key
			cmp = clientv3.Compare(clientv3.CreateRevision(i.key), "=", 0)
		} else {
			// update the key
			if id, err = strconv.ParseInt(string(kvs[0].Value), 10, 64); err != nil {
				return 0, errors.Trace(err)
			}
			cmp = clientv3.Compare(clientv3.Value(i.key), "=", string(kvs[0].Value))
		}

		id++

		if err = i.etcd.put(cmp, clientv3.OpPut(i.key, strconv.FormatInt(id, 10))); err != nil {
			if errors.Cause(err) == errLostMaster {
				continue
			}
			return 0, errors.Trace(err)
		}

		break
	}

	return id, nil
}
