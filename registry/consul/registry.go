package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/liuxp0827/grpc-lb/instance"
	"github.com/liuxp0827/grpc-lb/internal/logger"
	"github.com/liuxp0827/grpc-lb/registry"
	"sync"
	"time"
)

type consulRegistry struct {
	mu       sync.Mutex
	insts    map[string]*instance.Instance
	doneOnce sync.Once
	done     chan struct{}
	client   *api.Client
	wg       sync.WaitGroup
	logger   logger.Logger
}

func New(dc, addr string, l logger.Logger) (registry.Registry, error) {
	if dc == "" {
		dc = "dc1"
	}

	client, err := api.NewClient(&api.Config{
		WaitTime:   time.Second * 3,
		Datacenter: dc,
		Address:    addr,
	})
	if err != nil {
		return nil, err
	}

	if l == nil {
		l = logger.DefaultLogger
	}

	r := &consulRegistry{
		insts:  make(map[string]*instance.Instance),
		done:   make(chan struct{}),
		client: client,
		logger: l,
	}
	return r, nil
}

func (r *consulRegistry) Close() error {
	r.doneOnce.Do(func() {
		close(r.done)
		r.wg.Wait()
	})
	return nil
}

func (r *consulRegistry) Register(inst instance.Instance) <-chan error {
	errCh := make(chan error, 1)

	if dup := func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()

		addr := fmt.Sprintf("%s:%d", inst.Addr, inst.Port)

		_, dup := r.insts[addr]
		if dup {
			return dup
		}

		r.insts[addr] = &inst
		return false
	}; dup() {
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

		svcId := fmt.Sprintf("%s-%s-%d", inst.App, inst.Addr, inst.Port)
		if len(inst.Env) > 0 {
			svcId = fmt.Sprintf("%s-%s", inst.Env, svcId)
		}
		checkId := svcId
		svcName := inst.App
		if len(inst.Env) > 0 {
			svcName = fmt.Sprintf("%s/%s", inst.Env, svcName)
		}

		err := r.client.Agent().ServiceRegister(&api.AgentServiceRegistration{
			Kind:    api.ServiceKindTypical,
			ID:      svcId,
			Name:    svcName,
			Address: inst.Addr,
			Port:    inst.Port,
			Meta:    inst.Metadata.ToMap(),
			Check: &api.AgentServiceCheck{
				CheckID: checkId,
				TTL:     "10s",
			},
		})

		if err != nil {
			errCh <- err
			return
		}

		tick := time.NewTicker(time.Second * 6)
		defer tick.Stop()

		renewRetryTimes := 0
	loop:
		for {
			//log.Printf("heartbeat tick")
			select {
			case <-r.done:
				r.client.Agent().ServiceDeregister(svcId)
				errCh <- registry.ErrRegistryClosed
				break loop
			case <-tick.C:
				err := r.client.Agent().UpdateTTL(checkId, "pass", "pass")
				if err != nil {
					r.logger.Printf("[error] failed to update ttl, caused by: %s", err.Error())
					renewRetryTimes++
					if renewRetryTimes > registry.MaxRenewRetry {
						errCh <- registry.ErrFailedRenew
						break loop
					}
				}
			}
		}
	}()

	return errCh
}
