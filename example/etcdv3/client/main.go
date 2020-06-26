package main

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"github.com/liuxp0827/grpc-lb/example/proto"
	"github.com/liuxp0827/grpc-lb/internal/balancer/smooth_weighted"
	_ "github.com/liuxp0827/grpc-lb/resolver/etcdv3"
)

func main() {
	log.Println("client begin...")

	conn, err := grpc.Dial("etcd://127.0.0.1:2371,127.0.0.1:2372,127.0.0.1:2373/dev/demo", grpc.WithInsecure(),
		grpc.WithBalancerName(smooth_weighted.Name),
		grpc.WithBlock())
	if err != nil {
		log.Fatal(err)
	}

	client := proto.NewEchoSvcClient(conn)
	for {
		resp, err := client.Echo(context.Background(), &proto.EchoReq{})
		if err != nil {
			log.Println(err)
		} else {
			log.Println(resp.Msg)
		}
	}
}
