package exgrpc

import (
	"context"
	"fmt"
	pb "go-example/exgrpc/proto" // proto package
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func runClient() {
	// connect to grpc server
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	runClientExample(ctx, client)
}

func runClientExample(ctx context.Context, client pb.UserServiceClient) {
	// 1. Tạo một vài users
	fmt.Println("=== Creating Users ===")
	users := []*pb.CreateUserRequest{
		{Name: "Nguyễn Văn A", Email: "vana@example.com", Age: 25},
		{Name: "Trần Thị B", Email: "thib@example.com", Age: 30},
		{Name: "Lê Văn C", Email: "vanc@example.com", Age: 28},
	}

	for _, req := range users {
		res, err := client.CreateUser(ctx, req)
		if err != nil {
			log.Printf("Error creating user: %v", err)
			continue
		}
		fmt.Printf("Created: ID=%d, Name=%s, Email=%s\n",
			res.User.Id, res.User.Name, res.User.Email)
	}

	// 2. Lấy thông tin một user cụ thể
	fmt.Println("\n=== Get User By ID ===")
	getRes, err := client.GetUser(ctx, &pb.GetUserRequest{Id: 1})
	if err != nil {
		log.Printf("Error getting user: %v", err)
	} else {
		fmt.Printf("User: ID=%d, Name=%s, Email=%s, Age=%d\n",
			getRes.User.Id, getRes.User.Name, getRes.User.Email, getRes.User.Age)
	}

	// 3. Lấy danh sách tất cả users
	fmt.Println("\n=== List All Users ===")
	listRes, err := client.ListUser(ctx, &pb.ListUsersRequest{})
	if err != nil {
		log.Printf("Error listing users: %v", err)
	} else {
		fmt.Printf("Total users: %d\n", listRes.Total)
		for _, u := range listRes.Users {
			fmt.Printf("  - ID=%d, Name=%s, Email=%s, Age=%d\n",
				u.Id, u.Name, u.Email, u.Age)
		}
	}

	// 4. Stream users
	fmt.Println("\n=== Stream Users ===")
	stream, err := client.StreamUsers(ctx, &pb.StreamUsersRequest{})
	if err != nil {
		log.Printf("Error streaming users: %v", err)
		return
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error receiving stream: %v", err)
			break
		}
		fmt.Printf("Streamed: ID=%d, Name=%s\n",
			res.User.Id, res.User.Name)
	}

	fmt.Println("\n=== Done! ===")
}
