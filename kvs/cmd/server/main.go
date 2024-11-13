package main

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/paulja/gokvs/proto/clerk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func main() {
	fmt.Printf("go-kvs server listening\n")
	server := &Server{store: map[string]string{}}
	if err := server.Run(); err != nil {
		panic(err)
	}
}

type Server struct {
	mu    sync.Mutex
	store map[string]string

	clerk.UnimplementedClerkServiceServer
}

func (s *Server) Run() error {
	listen, err := net.Listen("tcp", ":4000")
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authIntercepter),
	)
	clerk.RegisterClerkServiceServer(grpcServer, s)
	return grpcServer.Serve(listen)
}

func authIntercepter(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	hander grpc.UnaryHandler,
) (
	interface{},
	error,
) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no headers in request")
	}
	headers, ok := md["authorization"]
	if !ok {
		return "", status.Error(codes.Unauthenticated, "no auth header in request")
	}
	if len(headers) != 1 {
		return "", status.Error(codes.Unauthenticated, "more than 1 auth header in request")
	}
	tokenctx := context.WithValue(ctx, "authtoken", headers[0])
	res, err := hander(tokenctx, req)
	return res, err
}

func (s *Server) Put(_ context.Context, req *clerk.PutRequest) (*clerk.PutResponse, error) {
	if len(req.Key) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "missing key")
	}
	s.mu.Lock()
	s.store[req.Key] = req.Value
	s.mu.Unlock()
	return &clerk.PutResponse{}, nil
}

func (s *Server) Append(
	_ context.Context,
	req *clerk.AppendRequest,
) (
	*clerk.AppendResponse,
	error,
) {
	if len(req.Key) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "missing key")
	}
	s.mu.Lock()
	value := s.store[req.Key]
	s.store[req.Key] = value + req.Arg
	s.mu.Unlock()
	return &clerk.AppendResponse{
		OldValue: value,
	}, nil
}

func (s *Server) Get(ctx context.Context, req *clerk.GetRequest) (*clerk.GetResponse, error) {
	token := ctx.Value("authtoken")
	fmt.Printf("token: %+v\n", token)

	if len(req.Key) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "missing key")
	}
	s.mu.Lock()
	value := s.store[req.Key]
	s.mu.Unlock()
	return &clerk.GetResponse{
		Value: value,
	}, nil
}
