package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"

	pb "github.com/serhatcetinkaya/grpc-demo-app/proto/math"

	"time"

	"google.golang.org/grpc"
)

var sleepTime = rand.Intn(4000) + 1000

func main() {
	rand.Seed(time.Now().Unix())

	var host = flag.String("h", "localhost", "Address of the server")
	var port = flag.Int("p", 50005, "Port of the server")
	flag.Parse()
	serverAddr := fmt.Sprintf("%s:%d", *host, *port)
	// dial server
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}

	// create stream
	client := pb.NewMathClient(conn)
	stream, err := client.Max(context.Background())
	if err != nil {
		log.Fatalf("openn stream error %v", err)
	}

	var max int32
	ctx := stream.Context()
	done := make(chan bool)

	// first goroutine sends random increasing numbers to stream
	go func() {
		for i := 1; i <= 20000000; i++ {
			// generate random nummber and send it to stream
			rnd := int32(rand.Intn(i))
			req := pb.Request{Num: rnd}
			if err := stream.Send(&req); err != nil {
				log.Fatalf("can not send %v", err)
			}
			log.Printf("%d sent", req.Num)
			time.Sleep(time.Second)
		}
		if err := stream.CloseSend(); err != nil {
			log.Println(err)
		}
	}()

	// second goroutine receives data from stream
	// and saves result in max variable
	//
	// if stream is finished it closes done channel
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(done)
				return
			}
			if err != nil {
				log.Fatalf("can not receive %v", err)
			}
			max = resp.Result
			log.Printf("new max %d received", max)
		}
	}()

	// third goroutine closes done channel
	// if context is done
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		close(done)
	}()

	<-done
	log.Printf("finished with max=%d", max)
}
