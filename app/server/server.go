package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	pb "github.com/serhatcetinkaya/grpc-demo-app/proto/math"

	"google.golang.org/grpc"
)

type server struct{}

type Host struct {
	IP   string `json:"ip_address"`
	Port int    `json:"port"`
	Tags struct {
		AZ       string `json:"az"`
		Canary   bool   `json:"canary"`
		LBWeight int    `json:"load_balancing_weight"`
	}
}

type Data struct {
	Hosts []Host `json:"hosts"`
}

var connCounter int32
var serverIP = os.Getenv("MY_IP")
var EDSServer = os.Getenv("EDS_SERVER")
var EDSServerEndpiont = fmt.Sprintf("http://%s:8080/edsservice/eds-cluster-service", EDSServer)
var isRegistered = false

func (s server) Max(srv pb.Math_MaxServer) error {

	atomic.AddInt32(&connCounter, 1)
	defer atomic.AddInt32(&connCounter, -1)
	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				log.Printf("Number of connections to the server: %d (pod IP: %v)", connCounter, serverIP)
				if connCounter < 3 {
					registerToEnvoyProxy()
				} else {
					deregisterFromEnvoyProxy()
				}
			}
		}
	}()

	var max int32
	ctx := srv.Context()

	for {

		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// receive data from stream
		req, err := srv.Recv()
		if err == io.EOF {
			// return will close stream from server side
			log.Println("exit")
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}

		// continue if number reveived from stream
		// less than max
		if req.Num <= max {
			continue
		}

		// update max and send it to stream
		max = req.Num
		resp := pb.Response{Result: max}
		if err := srv.Send(&resp); err != nil {
			log.Printf("send error %v", err)
		}
		log.Printf("send new max=%d", max)
	}
}

func registerToEnvoyProxy() {
	if isRegistered {
		return
	}
	var currentHosts Data

	var myself = make([]Host, 1)
	resp, err := http.Get(EDSServerEndpiont)
	if err != nil {
		fmt.Printf("error making http req: %q", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		myself[0].IP = serverIP
		myself[0].Port = 50005
		myself[0].Tags.AZ = "us-central1-a"
		myself[0].Tags.Canary = false
		myself[0].Tags.LBWeight = 50
		currentHosts.Hosts = append(currentHosts.Hosts, myself...)
		requestBody, err := json.Marshal(currentHosts)
		if err != nil {
			fmt.Printf("error json: %q", err)
		}

		timeout := time.Duration(5 * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		req, err := http.NewRequest("POST", EDSServerEndpiont, bytes.NewBuffer(requestBody))
		req.Header.Set("Content-type", "application/json")

		resp, err = client.Do(req)
		defer resp.Body.Close()
	} else if resp.StatusCode == http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&currentHosts)
		myself[0].IP = serverIP
		myself[0].Port = 50005
		myself[0].Tags.AZ = "us-central1-a"
		myself[0].Tags.Canary = false
		myself[0].Tags.LBWeight = 50
		currentHosts.Hosts = append(currentHosts.Hosts, myself...)
		requestBody, err := json.Marshal(currentHosts)
		if err != nil {
			fmt.Printf("error json: %q", err)
		}

		timeout := time.Duration(5 * time.Second)
		client := http.Client{
			Timeout: timeout,
		}
		req, err := http.NewRequest("PUT", EDSServerEndpiont, bytes.NewBuffer(requestBody))
		req.Header.Set("Content-type", "application/json")

		resp, err = client.Do(req)
		defer resp.Body.Close()
	}
	isRegistered = true
}

func deregisterFromEnvoyProxy() {
	var currentHosts Data
	resp, err := http.Get(EDSServerEndpiont)
	if err != nil {
		fmt.Printf("error making http req: %q", err)
	}
	if resp.StatusCode == http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&currentHosts)
		for i := len(currentHosts.Hosts) - 1; i >= 0; i-- {
			if currentHosts.Hosts[i].IP == serverIP {
				currentHosts.Hosts = append(currentHosts.Hosts[:i], currentHosts.Hosts[i+1:]...)
				requestBody, err := json.Marshal(currentHosts)
				if err != nil {
					fmt.Printf("error json: %q", err)
				}

				timeout := time.Duration(5 * time.Second)
				client := http.Client{
					Timeout: timeout,
				}
				req, err := http.NewRequest("PUT", EDSServerEndpiont, bytes.NewBuffer(requestBody))
				req.Header.Set("Content-type", "application/json")

				resp, err = client.Do(req)
				defer resp.Body.Close()
			}
		}
	}
	isRegistered = false
}

func main() {
	go func() {
		time.Sleep(5 * time.Second)
		log.Printf("registering to eds_service\n")
		registerToEnvoyProxy()
		log.Printf("registered to eds_service\n")
	}()

	// create listiner
	lis, err := net.Listen("tcp", ":50005")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// create grpc server
	s := grpc.NewServer()
	pb.RegisterMathServer(s, server{})

	// and start...
	log.Printf("start new server with ID: %v", serverIP)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
