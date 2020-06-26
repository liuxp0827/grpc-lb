# grpc-lb
grpc服务注册与发现，基于etcd/consul
- 服务注册
- 服务发现
- 平滑权重负载均衡

### instance
```go
type Metadata map[string]string

type App struct {
	Env      string   `json:"env"`
	Name     string   `json:"name"`
	Addr     string   `json:"addr"`
	Port     int      `json:"port"`
	Metadata Metadata `json:"metadata"`
}
```

### 服务注册
```go
r, _ := etcdv3.New(clientv3.Config{
	Endpoints:   []string{"127.0.0.1:2379"},
	DialTimeout: time.Second * 5,
})

// 执行异步注册，注册失败或者连续10次renew失败，直接返回error
errCh := r.Register(app.App{
	Env:      "dev",
	Name:      "demo",
	Addr:     "127.0.0.1",
	Port:     *port,
	Metadata: app.Metadata{"weight": strconv.Itoa(*weight)},
})

go func() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	select {
	case <-sig:
		r.Close()
	    s.GracefulStop()
	    os.Exit(0)
	case err := <-errCh:
		log.Fatalf("failed to register: %s", err.Error())
	}
}()
```

### 服务发现
target的格式：
```go
scheme://authority/endpoint
```
- scheme: 指定使用哪种服务发现实现，比如`etcd`或者`consul`
- authority: 指定etcd或者consul的连接地址
- endpoint: 指定要发现的服务名称

比如有个服务注册的`serviceName`为`app`，现在要使用`consul`实现，`consul`的地址为`127.0.0.1:8500`，则：
```go
import (
    _ "github.com/liuxp0827/grpc-lb/resolver/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

conn, err := grpc.Dial("consul://127.0.0.1:8500/dev/demo",
		grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name), grpc.WithBlock())
```
类似的，使用`etcd`实现的方法：
```go
import (
	_ "github.com/liuxp0827/grpc-lb/resolver/etcdv3"
	"github.com/liuxp0827/grpc-lb/internal/balancer/smooth_weighted"
    "google.golang.org/grpc"
)

// etcd的多个地址使用`,`隔开
conn, err := grpc.Dial("etcd://127.0.0.1:2379,127.0.0.1:2379,127.0.0.1:2379/dev/demo", grpc.WithInsecure(),
	grpc.WithBalancerName(smooth_weighted.Name),
	grpc.WithBlock())
```