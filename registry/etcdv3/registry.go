package etcdv3

import (
	"context"
	"fmt"
	"github.com/liuxp0827/grpc-lb/instance"
	"github.com/liuxp0827/grpc-lb/internal/logger"
	"github.com/liuxp0827/grpc-lb/registry"
	"go.etcd.io/etcd/clientv3"
	"path"
	"sync"
	"time"
)

// set prefix for key
func WithPrefix(prefix string) Option {
	return func(opts *Options) {
		opts.prefix = prefix
	}
}

func WithTTL(ttl int64) Option {
	return func(opts *Options) {
		opts.ttl = ttl
	}
}

func WithLogger(l logger.Logger) Option {
	return func(opts *Options) {
		opts.l = l
	}
}

type Option func(opts *Options)
type Options struct {
	ttl    int64
	prefix string
	l      logger.Logger
}

type Registry struct {
	mu       sync.Mutex
	insts    map[string]*instance.Instance
	doneOnce sync.Once
	done     chan struct{}
	client   *clientv3.Client
	opts     *Options
	wg       sync.WaitGroup
	logger   logger.Logger
}

func New(cfg clientv3.Config, opts ...Option) (registry.Registry, error) {

	client, err := clientv3.New(cfg)
	if err != nil {
		return nil, err
	}

	r := &Registry{
		insts:  make(map[string]*instance.Instance),
		done:   make(chan struct{}),
		opts:   new(Options),
		client: client,
	}

	for _, opt := range opts {
		opt(r.opts)
	}

	if r.opts.ttl <= 10 {
		r.opts.ttl = 10
	}

	if r.opts.prefix == "" {
		r.opts.prefix = "/grpc-discovery"
	}

	if r.opts.l == nil {
		r.opts.l = logger.DefaultLogger
	}

	return r, nil
}

func (r *Registry) Register(inst instance.Instance) <-chan error {
	errCh := make(chan error, 1)

	if dup := func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()

		addr := fmt.Sprintf("%s:%d", inst.Addr, inst.Port)

		_, dup := r.insts[addr]
		if dup {
			return true
		}
		r.insts[addr] = &inst

		return false
	}(); dup {
		errCh <- registry.ErrDupRegister
		return errCh
	}

	select {
	case <-r.done:
		errCh <- registry.ErrRegistryClosed
		return errCh
	default:
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		key := path.Join(r.opts.prefix, inst.Env, inst.App, fmt.Sprintf("%s:%d", inst.Addr, inst.Port))
		val := inst.Encode()
		cctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		lease, err := r.client.Grant(cctx, r.opts.ttl)
		cancel()
		if err != nil {
			errCh <- err
			return
		}

		cctx, cancel = context.WithTimeout(context.Background(), time.Second*3)
		_, err = r.client.Put(cctx, key, val, clientv3.WithLease(lease.ID))
		cancel()
		if err != nil {
			errCh <- err
			return
		}

		ticker := time.NewTicker(time.Duration(r.opts.ttl*2/3) * time.Second)
		defer ticker.Stop()

		renewRetryTimes := 0
	loop:
		for {
			select {
			case <-r.done:
				r.client.Delete(context.Background(), key)
				errCh <- registry.ErrRegistryClosed
				break loop
			case <-ticker.C:
				_, err := r.client.KeepAliveOnce(context.Background(), lease.ID)
				if err != nil {
					r.logger.Printf("[error] failed to update ttl, caused by: %s", err.Error())
					renewRetryTimes++
					// 如果续租失败达到一定次数，认为分区了，这时候程序应终止
					if renewRetryTimes > registry.MaxRenewRetry {
						errCh <- registry.ErrFailedRenew
						break loop
					}
				} else {
					renewRetryTimes = 0
				}
			}
		}
	}()

	return errCh
}

func (r *Registry) Close() error {
	r.doneOnce.Do(func() {
		close(r.done)
		r.wg.Wait()
		// 等待所有子协程都退出才关闭client连接
		r.client.Close()
	})
	return nil
}
