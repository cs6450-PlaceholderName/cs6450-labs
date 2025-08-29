package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rstutsman/cs6450-labs/kvs"
)

type Client struct {
	rpcClient *rpc.Client
}

func Dial(addr string) *Client {
	rpcClient, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	return &Client{rpcClient}
}

// Send a batch of keys to retrieve synchronously
func (client *Client) Get_Synch_Batch(keys []string) []string {
	request := kvs.Get_Batch_Request{
		Keys: keys,
	}
	response := kvs.Get_Batch_Response{}
	err := client.rpcClient.Call("KVService.Get_Batch", &request, &response)
	if err != nil {
		log.Fatal(err)
	}

	return response.Values
}

// Send a batch of key-value pairs to modify synchronously
func (client *Client) Put_Synch_Batch(putData map[string]string) {
	request := kvs.Put_Batch_Request{
		Data: putData,
	}
	//response := kvs.Put_Batch_Response{}
	err := client.rpcClient.Call("KVService.Put_Batch", &request, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// Asynchronous Get and Put counterparts
func (client *Client) Get_Asynch_Batch(keys []string) *rpc.Call {
	request := kvs.Get_Batch_Request{
		Keys: keys,
	}
	response := kvs.Get_Batch_Response{}

	call := client.rpcClient.Go("KVService.Get_Batch", &request, &response, nil)
	if call.Error != nil {
		log.Fatal(call.Error)
	}

	return call
}

func (client *Client) Put_Asynch_Batch(putData map[string]string) *rpc.Call {
	request := kvs.Put_Batch_Request{
		Data: putData,
	}
	response := kvs.Put_Batch_Response{}

	call := client.rpcClient.Go("KVService.Put_Batch", &request, &response, nil)
	if call.Error != nil {
		log.Fatal(call.Error)
	}

	return call
}

func runClient(id int, addr string, done *atomic.Bool, workload *kvs.Workload, resultsCh chan<- uint64, asynch bool) {
	client := Dial(addr)

	value := strings.Repeat("x", 128)
	const batchSize = 1024

	opsCompleted := uint64(0)

	for !done.Load() {
		// Create two batches of operations for reads and writes
		var getKeys []string
		putData := make(map[string]string)

		for j := 0; j < batchSize; j++ {
			op := workload.Next()
			key := fmt.Sprintf("%d", op.Key)
			if op.IsRead {
				//client.Get(key)
				getKeys = append(getKeys, key)
			} else {
				//client.Put(key, value)
				putData[key] = value
			}
			opsCompleted++
		}

		// Handle calls diferently based on RPC flavor (synchronous vs asynchronous)
		if asynch {
			var calls []*rpc.Call

			// Send only 2 RPC calls, each a batch
			if len(getKeys) > 0 {
				calls = append(calls, client.Get_Asynch_Batch(getKeys))
			}
			if len(putData) > 0 {
				calls = append(calls, client.Put_Asynch_Batch(putData))
			}

			// Wait for all asynchronous calls to complete.
			// Similar to what we did in HW1.
			// call.Done is a channel which signals when the call was finished
			// Response data is stored in call.Reply (NEEDS SOME TESTING)
			for _, call := range calls {
				<-call.Done
			}
		} else {
			// Send only 2 RPC calls, each a batch
			if len(getKeys) > 0 {
				client.Get_Synch_Batch(getKeys)
			}
			if len(putData) > 0 {
				client.Put_Synch_Batch(putData)
			}
		}
	}

	fmt.Printf("Client %d finished operations.\n", id)

	resultsCh <- opsCompleted
}

type HostList []string

func (h *HostList) String() string {
	return strings.Join(*h, ",")
}

func (h *HostList) Set(value string) error {
	*h = strings.Split(value, ",")
	return nil
}

func main() {
	hosts := HostList{}

	flag.Var(&hosts, "hosts", "Comma-separated list of host:ports to connect to")
	theta := flag.Float64("theta", 0.99, "Zipfian distribution skew parameter")
	workload := flag.String("workload", "YCSB-B", "Workload type (YCSB-A, YCSB-B, YCSB-C)")
	secs := flag.Int("secs", 30, "Duration in seconds for each client to run")

	// Change this value to run asynchronously
	asynch := flag.Bool("asynch", false, "Enable asynchronous RPC calls")

	flag.Parse()

	if len(hosts) == 0 {
		hosts = append(hosts, "localhost:8080")
	}

	fmt.Printf(
		"hosts %v\n"+
			"theta %.2f\n"+
			"workload %s\n"+
			"secs %d\n"+
			"Asynch RPC %t\n",
		hosts, *theta, *workload, *secs, *asynch,
	)

	start := time.Now()

	done := atomic.Bool{}
	resultsCh := make(chan uint64)

	host := hosts[0]
	clientId := 0
	go func(clientId int) {
		workload := kvs.NewWorkload(*workload, *theta)
		runClient(clientId, host, &done, workload, resultsCh, *asynch)
	}(clientId)

	time.Sleep(time.Duration(*secs) * time.Second)
	done.Store(true)

	opsCompleted := <-resultsCh

	elapsed := time.Since(start)

	opsPerSec := float64(opsCompleted) / elapsed.Seconds()
	fmt.Printf("throughput %.2f ops/s\n", opsPerSec)
}
