![Brimston (1)](https://github.com/fabricekabongo/loggerhead/assets/4486484/5d1c7777-ccce-44a5-bc5f-f2e5de23d96f)

# Loggerhead

## Path to 0.1.0

### TODO

- [X] Reduce chatter in the clustering protocol and prevent the DB from saturating the network with messages. Planned for `0.0.2`
- [ ] Improve the usage of Prometheus. Planned for `0.0.2`
- [ ] Implement subscription to Polygon's updates. Planned for `0.0.3`
- [ ] Use real ADSB traffic (I'm thinking a week's worth of global traffic) as data to run realistic benchmark `0.0.4`
- [ ] Offer the ability to enable RAFT for a cluster instead of just Gossip for consistency between nodes (slower). Planned for `0.1.0`
- [ ] Offer the ability to shard namespaces by TreeNodes with primary and replication across nodes (basically multiple RAFT running in parallel) `0.2.0`
- [ ] Implement storing and recovering state from disk. Planned for `0.3.0`

### Done
- [X] Connect the query language to the database
- [X] Connect the network interface to the database through the query processor
- [X] Implement the memory storage using a quadtree
- [X] Implement Benchmark for the storage
- [X] Implement the network interface
- [X] Implement the query language
- [X] Implement the clustering
- [X] Implement the Prometheus metrics
- [X] Implement the admin interface
- [X] Implement the gossip protocol

Loggerhead is a geospatial in-memory database built in Go. It is designed to be fast, efficient and to be used in a distributed environment
like Kubernetes.

It uses a gossip-based membership system and performs a best-effort synchronization of the nodes.

## Usage

The database exposes multiple ports for different purposes:

- 19998: for Reads queries.
- 19999: for Writes queries.
- 20000: for HTTP to consume metrics(Prometheus) and the admin interface where you can visualize the state of the cluster
- 20001: for the gossip protocol to communicate with other nodes

## Configuration

The database can be configured using environment variables or command line arguments.
So far, only two configurations are supported:

- `CLUSTER_DNS`: the DNS name of the cluster; this is used to discover other nodes in the cluster by extracting the
  IP addresses from the DNS record. This is very convenient for Kubernetes; you provide the service DNS, e.g., loggerhead.default.svc.cluster.local, and the database will find the other nodes quickly. If you scale up, the new nodes will join the cluster automatically).
- `MAX_CONNECTIONS`: This is the number of connections you want to allow per port (each for READ and PORT). You need to find the right balance between too few, which creates congestion on the operations per CPU Core (although it can handle quite a lot), and too many, which risks making the CPU go to 100% and slow the system. The idea here is that the database will be called by your backend services, so there is no need to allow too many connections. If you plan to open many connections, you must modify ulimit in Linux.
- `SEED_NODES`: (coming soon): a list of seed nodes to bootstrap the cluster.

## Building

The database requires 1.22.1 and GCC to build.

```shell
CGO_ENABLED=1 GOARCH=$TARGETARCH go build -o loggerhead
```

## Running

```shell

./loggerhead --cluster-dns=loggerhead.default.svc.cluster.local

```

The output will look like this:

```
2024/06/10 01:44:07 Please set the following environment variables:
2024/06/10 01:44:07 CLUSTER_DNS
2024/06/10 01:44:07 Reverting to flags...
2024/06/10 01:44:07 [DEBUG] memberlist: Initiating push/pull sync with:  [::1]:20001
2024/06/10 01:44:07 [DEBUG] memberlist: Stream connection from=[::1]:42194
2024/06/10 01:44:07 Sharing local state to a new node
2024/06/10 01:44:07 Sharing local state to a new node
2024/06/10 01:44:07 [DEBUG] memberlist: Initiating push/pull sync with:  172.45.0.2:20001
2024/06/10 01:44:07 Sharing local state to a new node
2024/06/10 01:44:07 [DEBUG] memberlist: Stream connection from=172.45.0.2:48278
2024/06/10 01:44:07 Sharing local state to a new node
2024/06/10 01:44:07 [DEBUG] memberlist: Initiating push/pull sync with:  172.45.0.2:20001
2024/06/10 01:44:07 Sharing local state to a new node
2024/06/10 01:44:07 [DEBUG] memberlist: Stream connection from=172.45.0.2:48282
2024/06/10 01:44:07 Sharing local state to a new node
===========================================================
Starting the Database Server
Cluster DNS:  loggerhead.default.svc.cluster.local
Use the following ports for the following services:
Writing location update: 19999
Reading location update: 19998
Admin UI (/) & Metrics(/metrics): 20000
Clustering: 20001
===========================================================

```

# Querying

The database supports the following queries: GET, SAVE, DELETE, and POLY (for polygon).

## READ
You must connect to port 19998 to read data from the database.

*Note that the "1.0" at the beginning of each message is the version of the format, should the format change in the future, I want the client to be able to identify that immediately.*

### GET
```shell
telnet localhost 19998
GET mynamespace myid

>> 1.0,mynamespace,myid,12.560000,13.560000
```

## POLY
```shell
telnet localhost 19998
POLY mynamespace 10.560000 10.560000 15.560000 15.560000
>> 1.0,mynamespace,myid,12.560000,13.560000
>> 1.0,mynamespace,myid2,12.560000,11.560000
>> 1.0,mynamespace,myid3,14.560000,13.560000
>> 1.0,done
```

## Writing
You will need to connect to port 19999 to write data to the database.

*try using short names for the namespace and id, as I use golang maps to store the data* and the maps are faster with short strings as keys.

## SAVE
```shell
telnet localhost 19999
SAVE mynamespace myid 12.560000 13.560000
>> 1.0,saved
```

## DELETE
```shell
telnet localhost 19999
DELETE mynamespace myid
>> 1.0,deleted
```


