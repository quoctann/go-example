package exgrpc

import (
	"flag"
	"log"
	"os"
)

/*

# Install dependencies:
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Gen code:
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    grpc/proto/user.proto

# After code gen:
proto/user.pb.go >> message definition
proto/user_grpc.pb.go >> service interface + client/server code

*/

func RunExample(skip bool) {
	if skip {
		return
	}
	// go run main.go >> start server
	if len(os.Args) < 2 {
		runServer()
		return
	}

	// go run main.go c >> start client
	flag.NewFlagSet("c", flag.ContinueOnError)
	if os.Args[1] != "c" {
		log.Println("Nothing run")
		return
	}

	runClient()
}
