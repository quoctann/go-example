package exgrpc

import (
	"context"
	"fmt"
	pb "go-example/exgrpc/proto" // proto package
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedUserServiceServer
	users  map[int32]*pb.User
	nextID int32
	mu     sync.Mutex
}

func runServer() {
	// tcp listener
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// create grpc server
	s := grpc.NewServer()
	pb.RegisterUserServiceServer(s, &server{
		users:  make(map[int32]*pb.User),
		nextID: 0,
	})

	fmt.Printf("gRPC running :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// Section implement use case

func (s *server) StreamUsers(req *pb.StreamUsersRequest, stream pb.UserService_StreamUsersServer) error {
	s.mu.Lock()
	users := make([]*pb.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	s.mu.Unlock()

	for _, user := range users {
		if err := stream.Send(&pb.UserResponse{
			User:    user,
			Message: "Streaming user",
		}); err != nil {
			return err
		}
		time.Sleep(time.Second) // Simulate streaming delay
	}

	return nil
}

func (s *server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	user := &pb.User{
		Id:    s.nextID,
		Name:  req.Name,
		Email: req.Email,
		Age:   req.Age,
	}

	s.users[s.nextID] = user
	log.Printf("Created user: %v", user)

	return &pb.UserResponse{
		User:    user,
		Message: "User created successfully",
	}, nil
}

func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[req.Id]
	if !exists {
		return &pb.UserResponse{
			Message: "User not found",
		}, nil
	}

	return &pb.UserResponse{
		User:    user,
		Message: "User found",
	}, nil
}

func (s *server) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	users := make([]*pb.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	return &pb.ListUsersResponse{
		Users: users,
		Total: int32(len(users)),
	}, nil
}
