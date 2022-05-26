package main

import (
	"fmt"
	"github.com/CryoCodec/jim/config"
	pb "github.com/CryoCodec/jim/internal/proto"
	serverImpl "github.com/CryoCodec/jim/server"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sockAddr := config.GetSocketAddress()

	fmt.Println("Clearing old socket instance")
	clearSocket(sockAddr)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received SIGTERM, exiting")
		clearSocket(sockAddr)
		os.Exit(1)
	}()

	fmt.Println("Starting up server...")
	listener, err := net.Listen(config.Protocol, sockAddr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully created server instance!")

	server := grpc.NewServer()
	jimImpl := serverImpl.CreateJimService()

	pb.RegisterJimServer(server, jimImpl)
	err = server.Serve(listener)
	if err != nil {
		log.Printf("Failed to start up server: %s", err)
	}
}

func clearSocket(sockAddr string) {
	if _, err := os.Stat(sockAddr); err == nil {
		if err := os.RemoveAll(sockAddr); err != nil {
			log.Fatal(err)
		}
	}
}
