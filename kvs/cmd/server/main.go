package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"

	"github.com/paulja/gokvs/proto/clerk"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	initOtel(ctx)
	fmt.Printf("go-kvs server listening\n")
	server := &Server{store: map[string]string{}}
	if err := server.Run(); err != nil {
		panic(err)
	}
}

func initOtel(ctx context.Context) {
	target := "collector:4317"

	// connect to the collector
	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}

	// create the data exporter for traces
	texp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithGRPCConn(conn),
	)
	if err != nil {
		log.Fatal(err)
	}
	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String("clerk-service"),
	))
	if err != nil {
		log.Fatal(err)
	}

	// create a trace provider to capture build our trace output
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithResource(res),
		tracesdk.WithSpanProcessor(tracesdk.NewBatchSpanProcessor(texp)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// create the data exporter for metrics
	mexp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
	)
	if err != nil {
		log.Fatal(err)
	}

	// create a metrics provider to build our metric output
	mp := metricsdk.NewMeterProvider(
		metricsdk.WithReader(metricsdk.NewPeriodicReader(mexp)),
		metricsdk.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// graceful shutdown of the providers
	go func() {
		<-ctx.Done()
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
		if err := mp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}()
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
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	clerk.RegisterClerkServiceServer(grpcServer, s)
	return grpcServer.Serve(listen)
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

func (s *Server) Get(_ context.Context, req *clerk.GetRequest) (*clerk.GetResponse, error) {
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
