package main

import (
	"fmt"

	"github.com/CryoCodec/jim/ipc"
)

func main() {
	fmt.Println("Starting up server...")
	server := ipc.CreateServer()
	fmt.Println("Successfully created server instance...")
	ipc.Listen(server)
}
