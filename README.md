# Execution instructions

ssh node[num] 
exit to leave

ip addr

./run-cluster.sh 1 1 "" "-asynch"

nproc - number of cores



# notes
runcluster script autmaitcally if give num servers will make the remaining one's clients. also as client arg it will give all the server ip:ports as hosts which ends up in hostlist.

# Results

Final throughput numbers
Some rough numbers on hardware utilization metrics (CPU, memory, network)
Scaling characteristics (how performance changes with cluster size and/or with increasing
offered client load)
At a minimum, if your approach scales run it with small scale and larger scale
Any performance graphs and visualizations for the above 

# Design

We implemented the following:

### Batching
Generating an RPC call per each Get/Put operation is costly and inefficient. Instead, we implemented request batching, where a batch of operations is sent in a single RPC call. This significantly reduces networking overhead, allowing for better performance both on clients and servers.
### Asynch RPC 
By default, client made RPC calls synchronously to the server, meaning that each request was waited upon until completion. Instead, we implemented asynchronous RPC calls, significantly boosting client throughput.  
### Sharding
todo
### Client-side parallelism
todo
### At least once scheme
todo


# Reproducibility

Step-by-step instructions to reproduce results
Hardware requirements and setup
Software dependencies and installation if anything more than go, etc
Configuration parameters and their effects in particular if you’ve added ”knobs”

# Reflections

What you learned from the assignment
What optimizations worked well and why
What didn’t work and lessons learned
Ideas for further improvement
A short note on individual contributions from each team member
