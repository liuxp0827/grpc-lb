package etcdv3

import (
	"github.com/liuxp0827/grpc-lb/internal/backoff"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc/resolver"
	"path"
	"strings"
	"time"
)

var (
	PathPrefix      = "/grpc-discovery"
	BackoffMaxDelay = time.Second * 1
	DialTimeout     = time.Second * 5
)

func init() {
	resolver.Register(&etcdBuilder{})
}

type etcdBuilder struct{}

// etcd://192.168.50.10:2379,192.168.50.11:2379,192.168.50.12:2379/dev/echo
func (b *etcdBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	key := path.Join(PathPrefix, target.Endpoint)
	r := &etcdResolver{
		cc:      cc,
		key:     key,
		done:    make(chan struct{}),
		backoff: backoff.New(BackoffMaxDelay).Backoff,
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(target.Authority, ","),
		DialTimeout: DialTimeout,
	})
	if err != nil {
		return nil, err
	}

	r.client = client

	go r.watch()

	return r, nil
}

func (b *etcdBuilder) Scheme() string {
	return "etcd"
}
