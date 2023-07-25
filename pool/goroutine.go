package pool

import (
	"fmt"
	"github.com/davveo/lemonShop-framework/logger"
	"github.com/panjf2000/ants/v2"
	"sync"
	"time"
)

var commonPool CommonPool
var commonPoolOnce sync.Once
var workerPool WorkerPool
var workerPoolOnce sync.Once

type CommonPool struct {
	*ants.Pool
}

type WorkerPool struct {
	*ants.PoolWithFunc
}

type Conf struct {

	// Size is the size of goroutine pool.
	Size int

	// ExpiryDuration is a period for the scavenger goroutine to clean up those expired workers,
	// the scavenger scans all workers every `ExpiryDuration` and clean up those workers that haven't been
	// used for more than `ExpiryDuration`.
	ExpiryDuration time.Duration

	// PreAlloc indicates whether to make memory pre-allocation when initializing Pool.
	PreAlloc bool

	// Max number of goroutine blocking on pool.Submit.
	// 0 (default value) means no such limit.
	MaxBlockingTasks int

	// When Nonblocking is true, Pool.Submit will never be blocked.
	// ErrPoolOverload will be returned when Pool.Submit cannot be done at once.
	// When Nonblocking is true, MaxBlockingTasks is inoperative.
	Nonblocking bool

	// PanicHandler is used to handle panics from each worker goroutine.
	// if nil, panics will be thrown out again from worker goroutines.
	PanicHandler func(interface{})

	// When DisablePurge is true, workers are not purged and are resident.
	DisablePurge bool

	// PoolFunc is the function for processing tasks.
	PoolFunction func(interface{})
}

func InitWorkerPool(conf *Conf) error {
	var err error
	var poolWithFunc *ants.PoolWithFunc
	workerPoolOnce.Do(func() {
		poolWithFunc, err = ants.NewPoolWithFunc(conf.Size, conf.PoolFunction,
			ants.WithExpiryDuration(conf.ExpiryDuration),
			ants.WithPreAlloc(conf.PreAlloc),
			ants.WithMaxBlockingTasks(conf.MaxBlockingTasks),
			ants.WithNonblocking(conf.Nonblocking),
			ants.WithPanicHandler(conf.PanicHandler),
		)
	})
	if err != nil {
		logger.GLogger.Fatal(fmt.Sprintf("init go routine pool failed, err: %v", err))
		return err
	}
	workerPool = WorkerPool{poolWithFunc}
	return nil
}

func InitCommonPool(conf *Conf) error {
	var err error
	var pool *ants.Pool
	commonPoolOnce.Do(func() {
		pool, err = ants.NewPool(conf.Size,
			ants.WithExpiryDuration(conf.ExpiryDuration),
			ants.WithPreAlloc(conf.PreAlloc),
			ants.WithMaxBlockingTasks(conf.MaxBlockingTasks),
			ants.WithNonblocking(conf.Nonblocking),
			ants.WithPanicHandler(conf.PanicHandler),
		)
	})
	if err != nil {
		logger.GLogger.Fatal(fmt.Sprintf("init go routine pool failed, err: %v", err))
		return err
	}
	commonPool = CommonPool{pool}
	return nil
}

func GetCommonPool() *CommonPool {
	if commonPool.Pool == nil {
		logger.GLogger.Fatal("common go routine pool is nil")

	}
	return &commonPool
}

func GetWorkerPool() *WorkerPool {
	if workerPool.PoolWithFunc == nil {
		logger.GLogger.Fatal("common go routine pool is nil")

	}
	return &workerPool
}

func (p *CommonPool) Submit(task func()) error {
	return p.Pool.Submit(task)
}

func (p *WorkerPool) Invoke(args interface{}) error {
	return p.PoolWithFunc.Invoke(args)
}

func (p *CommonPool) Release() {
	p.Pool.Release()
}

func (p *WorkerPool) Release() {
	p.PoolWithFunc.Release()
}
