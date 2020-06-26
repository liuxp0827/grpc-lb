package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/liuxp0827/grpc-lb/app"
	"github.com/liuxp0827/grpc-lb/example/proto"
	"github.com/liuxp0827/grpc-lb/internal/logger"
	"github.com/liuxp0827/grpc-lb/registry/consul"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var InstanceID = os.Getenv("INSTANCE_ID")

type EchoServer struct{}

func (EchoServer) Echo(ctx context.Context, req *proto.EchoReq) (resp *proto.EchoResp, err error) {
	log.Printf("handle grpc req...")
	return &proto.EchoResp{
		Msg: InstanceID,
	}, err
}

func main() {
	port := flag.Int("port", 6060, "port")
	flag.Parse()

	r, err := consul.New("dc1", "http://127.0.0.1:8500", logger.DefaultLogger)
	if err != nil {
		log.Fatal(err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	proto.RegisterEchoSvcServer(s, &EchoServer{})

	errCh := r.Register(app.App{
		Env:  "dev",
		Name: "demo",
		Addr: "127.0.0.1",
		Port: *port,
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

	log.Printf("listening port: %d", *port)
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
