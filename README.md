# Useful commands

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

## Batching
todo
## Sharding
todo
## Client-side parallelism
todo
## Asynch RPC 
todo
## At least once scheme
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
