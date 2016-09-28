package main

import (
	"fmt"

	"github.com/abelnation/rts-server/comm"
)

func startServer() (*comm.Server, error) {
	gameServer, err := comm.Listen("127.0.0.1:8080")
	if err != nil {
		return nil, err
	}
	return gameServer, nil
}

func main() {

	// Start game server
	gameServer, err := startServer()
	if err != nil {
		fmt.Println("Error starting server: ", err);
		return
	}
	defer gameServer.Close()
	fmt.Println("Listening for connections...");

	// start server loop
	err = gameServer.Loop()

	fmt.Println("Server done")

}