# Results

## Final throughput numbers

1 client and 1 server: total `~8M op/s`

2 clients and 2 servers: total `~17M op/s`

4 clients and 4 servers: total `~33M op/s`

## Hardware utilization metrics

pprof results:

```
(pprof) top5
Showing nodes accounting for 109.42s, 26.16% of 418.28s total
Dropped 516 nodes (cum <= 2.09s)
Showing top 5 nodes out of 142
      flat  flat%   sum%        cum   cum%
    28.91s  6.91%  6.91%     37.66s  9.00%  runtime.findObject
    22.55s  5.39% 12.30%    149.53s 35.75%  encoding/gob.(*Decoder).decodeStruct
    20.08s  4.80% 17.10%     66.70s 15.95%  runtime.scanobject
    19.36s  4.63% 21.73%     19.36s  4.63%  internal/runtime/maps.ctrlGroup.matchH2
    18.52s  4.43% 26.16%     18.52s  4.43%  time.runtimeNow
(pprof) top5 -cum
Showing nodes accounting for 0.01s, 0.0024% of 418.28s total
Dropped 516 nodes (cum <= 2.09s)
Showing top 5 nodes out of 142
      flat  flat%   sum%        cum   cum%
     0.01s 0.0024% 0.0024%    169.15s 40.44%  net/rpc.(*service).call
         0     0% 0.0024%    158.26s 37.84%  net/http.(*conn).serve
         0     0% 0.0024%    158.25s 37.83%  net/http.(*ServeMux).ServeHTTP
         0     0% 0.0024%    158.25s 37.83%  net/http.serverHandler.ServeHTTP
         0     0% 0.0024%    158.25s 37.83%  net/rpc.(*Server).ServeHTTP
```

htop server sample during execution, showing fantastic CPU utilization:

![htop sample result](images/htop_node0.png)

## Scaling characteristics 

Our final throughput numbers (shown above) dempnstrate linear relationship between the number of machines and system throughput, showing that our solution scales well. This is achieved via sharding and high levels of parallelism.

# Design

## Successful Ideas
We implemented the following:
### Batching
Generating an RPC call per each Get/Put operation is costly and inefficient. Instead, we implemented request batching, where a batch of operations is sent in a single RPC call. This significantly reduces networking overhead, allowing for better performance both on clients and servers. We also found that a signficant amount of time was spent serializing and deserializing RPC payloads, calling handler functions, and most crucially spinning up a new goroutine per request (even over the same TCP connection). This operational logic was far more costly then the business logic which was often a lookup into the L1 cache. Additionally, we introduced logic for intelligent use of locks in individual RPC messages (as opposed to during the entire batch) for higher throughput between threads.
### Asynchronous RPC 
By default, clients made RPC calls synchronously to the server, meaning that each request was waited upon until completion. Instead, we implemented asynchronous RPC calls, significantly boosting client throughput. This removed head of line blocking between batches while not affecting linearizability from the servers POV, as the client gives up guarantees by not awaiting.
### Sharding
We sharded between and within machines. The client was responsible for consistently routing a key to the same server. Each server itself had many shards each with their own Reader Writer lock. This simulated a sharding + bucket lock design. The goal of this was to make each machine have roughly equal load and reduce lock contention within machines, so that the CPU was not idle. We implemented this after batching so the CPU was no longer hitting capactiy on small payloads because it didn't have to deal with RPC overhead.
### Client-side parallelism
Each client has multiple go routines pulling keys and sending batched request. This is required to maxamize CPU utilization on the client and servers (due to increased load). Just running multiple clients on multiple machines is not enough, so we ran experiments to optimize the number of clients per machine. One 16 core machine requires around 20 client routines to maximize CPU utilization and throughput. 
### At least once scheme
We implement an at-least-once delivery guarantee through a combination of client-side retry logic and server-side request deduplication:

**Client-side retry mechanism**: Each batch request is assigned a unique request ID using cryptographically secure random generation. If an RPC call fails, the client retries up to 3 times with exponential backoff (100ms, 200ms, 400ms delays). This ensures requests are delivered at least once, even in the presence of network failures.

**Server-side request deduplication**: The server maintains a request cache that maps request IDs to their corresponding responses. When processing a batch request, the server first checks if the request ID has been seen before. If found in the cache, it returns the cached response immediately without re-executing the operations. This prevents duplicate execution while maintaining the at-least-once guarantee.

The combination of unique request IDs, response caching, and retry logic ensures reliable message delivery without compromising data consistency, even when network partitions or server failures cause message loss or duplication.

## Linearizability

Given the previously described implementation details of, in particular, at-least-once semantics and batching, it becomes easy to recognize our key-value store as linearizable.
In particular we recognize the linearization point as being a real element of the following common sequence:
1. A client queues an RPC message into a batch message
    - Note that this is done *in order* of receipt- that is to say that the batch message preserves the sequence of individual RPC messages.
2. The batch message crosses the network to its corresponding shard. The client always receives a response, this is guaranteed by at-least-once semantics.
4. In handling the batch message, the server handles each individual RPC message in the order preserved by the batch message.
  - Effectively, each finalized RPC operation handled in this order guarantees a sequential history of a given object per batch message, reflected in the "batch response" from the server.
6. In handling the individual RPC message, the worker routine locks relevant data structures and performs the requested action.
   - **This locking of data structures** is considered the linearization point. Every action done when the lock is held is effectively atomic.
   - This still linearizes in instances where multiple batch messages affecting the same data are processed concurrently due to the atomic guarantees of the lock.

In essense, because we know that the client always knows the "timeframe" in which the message is in transit due to at-least-once semantics, and that a linearization point exists for each individual RPC message,
we know that this system is linearizable by definition.

## Failed Ideas
We implemented the following without getting performance improvments:
### Fine-grained locking
Storing a (value, RWlock) tuple to have fine grained key locking had a 2x slowdown because the extra controlflow was signficiantly more expensive than the business operations. 

### Worker Threads
We attempted to maintain a pool of workers and distribute. batch workloads among them via channels but found serializable execution to be as fast if not faster. Although having a pool solved the problem of goroutine creation overhead, switching on each action and routing to the correct channel and then awaiting in the case of a put on a temporary channel or state proved costly. 

### Shard-agnostic batches
Batching across different shards is costly by requiring a server-side proxy to redistribute per batch. We place the responsibility on client processes to organize individual RPC messages into shard-specific batch messages. For the purposes of this project, this is arguably simpler to reason about, as there are less moving parts, but more sophisticated schemas may opt for the proxy-reliant approach. This is usually due to simplicity of client implementation etc.

# Reproducibility
Our experiments were done on 8 CloudLab m510 machines.

run `./run-cluster.sh <server_count> <client_count> "-numshards <#>" "-asynch=<True/False> -connections <#>"`

For example, for best results:

`./run-cluster.sh 4 4 "-numshards 1000" "-asynch=True -connections 70"`

The number of shards is per server, so in total you will have `server_count * numshards`. 

`-connections` is how many goroutines each client machine runs. For optimal performance we suggest doing an exponential grid search followed by linear finetuning.

`-asynch` flag enables asynchronous RPC calls. Choose `true` for best performance.

`-connections` flag defines the number of parallel threads per client. 

# Reflections

Peeling back the onion is an invariant of optimizing. Being able to understand typical latencies for network and machine operations and in what cases they occur allow you to measure what matters. This should be done before optimizing as it is often less time consuming and pays dividends. But, just cause you see opportunity—low cpu usage, bad IPC, cache misses, high cost functions, blocking, lock contention—doesn't mean it will be easy to fix as the solutions come with tradeoffs or unexpected consequences. And sometimes the Usual Suspects drive you up the wrong wall.

We also believe the usual notion of "implement it simply first and then optimize." Having a correct implementation from the beginning can help distinguish from incorrect implementations during optimization. It also allows for more focused work, reliant on observations of bottlenecks.

Individual contributions from each team member:

- Artem: batching, asynch RPC
- Ash: "per-shard", unified RPC message batching, server-side sharding, "intelligent use of" locks
- Brendon: client side concurrency, sharding, failed ideas
- Cayden: at-least-once semantics (request IDs and request cache/deduplication), general fixes and refactoring
