package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/liuxp0827/grpc-lb/internal/backoff"
	"github.com/liuxp0827/grpc-lb/internal/logger"
	"google.golang.org/grpc/resolver"
	"sync"
	"time"
)

type Watcher struct {
	addresses chan []resolver.Address
	client    *api.Client
	dc        string // DataCenter
	key       string // ServiceName
	done      chan struct{}
	doneOnce  sync.Once
	logger    logger.Logger
	backoff   func(int) time.Duration
}

func NewWatcher(addr, srvName string) (*Watcher, error) {

	dc := DataCenter
	client, err := api.NewClient(&api.Config{
		Datacenter: dc,
		Address:    addr,
		Scheme:     "http",
	})
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		addresses: make(chan []resolver.Address, 0),
		client:    client,
		dc:        dc,
		key:       srvName,
		done:      make(chan struct{}),
		backoff:   backoff.New(BackoffMaxDelay).Backoff,
		doneOnce:  sync.Once{},
		logger:    logger.DefaultLogger,
	}

	go watcher.watch()

	return watcher, nil
}

func (w *Watcher) watch() {
	qo := &api.QueryOptions{
		Datacenter: w.dc,
		WaitTime:   time.Second * 10,
	}

	retryTimes := 0

	for {
		addrs, qm, err := w.client.Health().Service(w.key, "", false, qo)
		if err != nil {
			w.logger.Printf("[error]failed to resolve addr, caused by %s", err)
			delay := w.backoff(retryTimes)
			retryTimes++
			time.Sleep(delay)
			continue
		}

		if w.hasClosed() {
			break
		}

		qo.WaitIndex = qm.LastIndex

		addresses := make([]resolver.Address, len(addrs))

		for i := range addrs {
			svc := addrs[i].Service
			addresses[i] = resolver.Address{
				Addr:       fmt.Sprintf("%s:%d", svc.Address, svc.Port),
				ServerName: svc.Service,
			}
			addresses[i].Metadata = &svc.Meta
		}

		if w.hasClosed() {
			break
		}

		w.Notify(addresses)

		retryTimes = 0
	}
}

func (w *Watcher) Watch() <-chan []resolver.Address {
	return w.addresses
}

func (w *Watcher) Notify(addresses []resolver.Address) {
	w.addresses <- addresses
}

func (w *Watcher) hasClosed() bool {
	select {
	case <-w.done:
		return true
	default:
	}
	return false
}

func (w *Watcher) Close() {
	w.doneOnce.Do(func() {
		close(w.done)
	})
}
