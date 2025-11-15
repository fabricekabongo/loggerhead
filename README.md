![Brimston (1)](https://github.com/fabricekabongo/loggerhead/assets/4486484/5d1c7777-ccce-44a5-bc5f-f2e5de23d96f)
[![DeepSource](https://app.deepsource.com/gh/fabricekabongo/loggerhead.svg/?label=active+issues&show_trend=true&token=y2MpvgmywVPyLIUiutUfCDve)](https://app.deepsource.com/gh/fabricekabongo/loggerhead/)

# Loggerhead


Loggerhead is a geospatial in-memory database built in Go. It is designed to be fast, efficient and to be used in a distributed environment
like Kubernetes.

It uses a gossip-based membership system and performs a best-effort synchronization of the nodes.

## Benchmark of the core world engine

I ran the benchmark for 2 seconds, with the engine running on 1,2,4,8,16 and 32 cores. 
I will display here only performance on 1,2, and 4 cores. 
I also run the test single threaded (1 request at a time, more wait) and multithreaded (as many request as the core can handle sending, more chance of read/write locks).
This tests the core engine of the database, it doesn't test the connection as this add more variable based on the environment.

Expect slight decrease in performance when using with real network.

### Engine running on 1 Core

#### Single Threaded Test

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 2,674,922      | 1,931          | 214          | 1           |
| GetLocation          | Return a single location              | 11,557,712     | 214.6          | 7            | 0           |
| GetLocationsInRadius | Locations in Singapore (~734.3 km²)   | 3,264,337      | 748.2          | 44           | 1           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 586,419        | 28,216         | 8,198        | 8           |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 1,310          | 8,521,009      | 2,612,440    | 1,154       |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 1,027          | 9,785,152      | 3,176,647    | 1,331       |
| Delete               | Delete a location                     | 50,617,816     | 49.44          | 7            | 0           |

#### MultiThreaded Test

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 2,691,468      | 1,982          | 216          | 1           |
| GetLocation          | Return a single location              | 54,935,690     | 41.60          | 0            | 0           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 10,000         | 212,897        | 85,952       | 83          |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 100            | 82,167,521     | 26,660,673   | 11,594      |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 100            | 99,728,196     | 32,027,345   | 13,281      |
| Delete               | Delete a location                     | 54,722,920     | 46.31          | 7            | 0           |

### Engine running on 2 cores

#### Single Threaded Test

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 2,579,481      | 1,581          | 191          | 1           |
| GetLocation          | Return a single location              | 11,384,558     | 201.7          | 7            | 0           |
| GetLocationsInRadius | Locations in Singapore (~734.3 km²)   | 3,149,793      | 769.3          | 28           | 1           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 467,748        | 12,260         | 8,080        | 8           |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 1,318          | 4,504,925      | 2,515,047    | 1,146       |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 1,148          | 5,181,418      | 2,998,856    | 1,313       |
| Delete               | Delete a location                     | 50,270,070     | 48.34          | 7            | 0           |

#### Multi Threaded Test

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 3,910,928      | 1,129          | 130          | 1           |
| GetLocation          | Return a single location              | 28,343,620     | 84.74          | 0            | 0           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 37,132         | 77,616         | 58,016       | 78          |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 100            | 23,398,650     | 18,263,379   | 10,474      |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 100            | 30,380,619     | 21,837,460   | 12,006      |
| Delete               | Delete a location                     | 48,852,799     | 50.14          | 7            | 0           |


### Engine running on 4 cores

#### Single Threaded Test

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 1,000,000      | 1,377          | 358          | 2           |
| GetLocation          | Return a single location              | 7,754,721      | 154.1          | 7            | 0           |
| GetLocationsInRadius | Locations in Singapore (~734.3 km²)   | 1,863,948      | 644.2          | 5            | 0           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 560,006        | 3,492          | 2,668        | 3           |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 2,576          | 855,589        | 718,302      | 317         |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 1,528          | 1,295,831      | 940,770      | 359         |
| Delete               | Delete a location                     | 25,431,978     | 48.35          | 7            | 0           |

#### Multi Threaded Test

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 3,282,406      | 803.2          | 53           | 1           |
| GetLocation          | Return a single location              | 18,874,444     | 63.33          | 0            | 0           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 50,386         | 23,041         | 24,008       | 33          |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 208            | 6,089,438      | 6,361,179    | 3,143       |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 174            | 6,960,838      | 7,979,068    | 3,525       |
| Delete               | Delete a location                     | 23,993,090     | 49.35          | 7            | 0           |



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
## Path to 0.1.0

### TODO

- [ ] Implement subscription to Polygon's updates. Planned for `0.0.4`
- [ ] Use real ADSB traffic (I'm thinking a week's worth of global traffic) as data to run realistic benchmark `0.0.5`
- [ ] Offer the ability to enable RAFT for a cluster instead of just Gossip for consistency between nodes (slower). Planned for `0.1.0`
- [ ] Offer the ability to shard namespaces by TreeNodes with primary and replication across nodes (basically multiple RAFT running in parallel) `0.2.0`
- [ ] Implement storing and recovering state from disk. Planned for `0.3.0`

### Done

- [X] Improve the usage of Prometheus. Planned for `0.0.3`
- [X] Reduce chatter in the clustering protocol and prevent the DB from saturating the network with messages. Planned for `0.0.2`
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

