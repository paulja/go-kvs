package main

import (
	"context"
	"fmt"
	"os"

	"github.com/paulja/gokvs/proto/clerk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {

	if len(os.Args) < 2 {
		usage("no enough arguments")
	}

	switch os.Args[1] {
	case "put":
		if len(os.Args) != 4 {
			usage("PUT missing arguments")
		}
		fmt.Printf("PUT(%s, %s)\n", os.Args[2], os.Args[3])
		c := makeClient()
		_, err := c.Put(context.Background(), &clerk.PutRequest{
			Key:   os.Args[2],
			Value: os.Args[3],
		})
		if err != nil {
			panic(err)
		}
	case "append":
		if len(os.Args) != 4 {
			usage("APPEND missing arguments")
		}
		fmt.Printf("APPEND(%s, %s)\n", os.Args[2], os.Args[3])
		c := makeClient()
		r, err := c.Append(context.Background(), &clerk.AppendRequest{
			Key: os.Args[2],
			Arg: os.Args[3],
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("%+v\n", r)
	case "get":
		if len(os.Args) != 3 {
			usage("GET missing arguments")
		}
		fmt.Printf("GET(%s)\n", os.Args[2])
		c := makeClient()
		r, err := c.Get(context.Background(), &clerk.GetRequest{
			Key: os.Args[2],
		})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Value:%q\n", r.Value)
	}
}

func usage(msg string) {
	fmt.Println(msg)
	fmt.Printf("usage: \n\tput [key] [val]\n\tappend [key] [arg]\n\tget [key]\n")
	os.Exit(2)
}

func makeClient() clerk.ClerkServiceClient {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.NewClient(":4000", opts...)
	if err != nil {
		panic(err)
	}
	return clerk.NewClerkServiceClient(conn)
}
