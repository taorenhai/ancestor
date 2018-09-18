package pd

/*
import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
)


var (
	ida *idAllocator
)

func init() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379", "127.0.0.1:22379", "127.0.0.1:32379"},
		DialTimeout: time.Minute,
	})

	if err != nil {
		fmt.Printf("clientv3.New error:%s\n", err.Error())
		os.Exit(-1)
	}

	ida = newIDAllocator(cli)
}

func TestID(t *testing.T) {
	ids := make(map[int64]struct{})

	for i := 0; i < 100; i++ {
		id, err := ida.newID()
		if err != nil {
			t.Fatalf("ida newID error:%s", err.Error())
		}
		if _, ok := ids[int64(id)]; ok {
			t.Fatalf("ida find old id:%d", id)
		}
		ids[int64(id)] = struct{}{}
	}

}

func TestMultiThread(t *testing.T) {
	ids := make(map[int64]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 10; j++ {
				id, err := ida.newID()
				if err != nil {
					t.Errorf("ida newID error:%s", err.Error())
					os.Exit(-1)
				}
				mu.Lock()
				if _, ok := ids[int64(id)]; ok {
					t.Errorf("ida find old id:%d", id)
					os.Exit(-1)
				}
				ids[int64(id)] = struct{}{}
				mu.Unlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
*/
