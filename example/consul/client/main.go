package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"log"
	"github.com/liuxp0827/grpc-lb/example/proto"
	_ "github.com/liuxp0827/grpc-lb/resolver/consul"
	"time"
)

func main() {

	// target: scheme://authority/endpoint
	//      - scheme: `etcd`则使用etcd实现，`consul`则使用consul实现
	//      - authority: 指定etcd或者consul的连接地址
	//      - endpoint: 指定服务路径，服务端注册时的服务名
	conn, err := grpc.Dial("consul://127.0.0.1:8500/dev/demo",
		grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name), grpc.WithBlock())
	if err != nil {
		log.Fatal(err)
	}

	client := proto.NewEchoSvcClient(conn)
	for {
		time.Sleep(time.Second)
		resp, err := client.Echo(context.Background(), &proto.EchoReq{})
		if err != nil {
			log.Println(err)
		} else {
			log.Println(resp.Msg)
		}
	}
}
