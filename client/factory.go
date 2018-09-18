package client

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/zssky/log"
)

const (
	maxWorkerSize = 100
)

type task struct {
	call func(w *worker)
}

type worker struct {
	name    string
	factory *factory
}

type factory struct {
	taskChan chan *task
	wg       sync.WaitGroup
}

func (f *factory) asyncRun(call func(w *worker)) {
	f.taskChan <- &task{call: call}

}

func newFactory() *factory {
	return &factory{
		taskChan: make(chan *task, maxWorkerSize*2),
	}
}

func (f *factory) start() {
	for i := 0; i < maxWorkerSize; i++ {
		w := &worker{name: fmt.Sprintf("%.4d", i), factory: f}
		f.wg.Add(1)
		go w.start()
	}
}

func (f *factory) stop() {
	for i := 0; i < maxWorkerSize; i++ {
		f.asyncRun(func(w *worker) {
			runtime.Goexit()
		})
	}
	log.Debugf("wait worker stop")
	f.wg.Wait()
	log.Debugf("stop success")
}

func (w *worker) start() {
	defer func() {
		w.factory.wg.Done()
	}()
	for {
		task := <-w.factory.taskChan
		task.call(w)
	}
}
