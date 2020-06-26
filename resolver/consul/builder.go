package consul

import (
	"github.com/hashicorp/consul/api"
	"github.com/liuxp0827/grpc-lb/internal/backoff"
	"google.golang.org/grpc/resolver"
	"time"
)

var (
	DataCenter      = "dc1"
	BackoffMaxDelay = time.Second * 1
)

func init() {
	resolver.Register(&consulBuilder{
	})
}

type consulBuilder struct {
	addresses chan []resolver.Address
}

func (b *consulBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	dc := DataCenter
	client, err := api.NewClient(&api.Config{
		Datacenter: dc,
		Address:    target.Authority,
		Scheme:     "http",
	})
	if err != nil {
		return nil, err
	}

	r := &consulResolver{
		cc:      cc,
		client:  client,
		dc:      dc,
		key:     target.Endpoint,
		done:    make(chan struct{}),
		backoff: backoff.New(BackoffMaxDelay).Backoff,
	}

	go r.watch()

	return r, nil
}

func (b *consulBuilder) Scheme() string {
	return "consul"
}
