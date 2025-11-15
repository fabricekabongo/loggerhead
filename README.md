![Brimston (1)](https://github.com/fabricekabongo/loggerhead/assets/4486484/5d1c7777-ccce-44a5-bc5f-f2e5de23d96f)
[![DeepSource](https://app.deepsource.com/gh/fabricekabongo/loggerhead.svg/?label=active+issues\&show_trend=true\&token=y2MpvgmywVPyLIUiutUfCDve)](https://app.deepsource.com/gh/fabricekabongo/loggerhead/)

# Loggerhead

**Loggerhead is a geospatial in-memory database for fast location lookups and area queries.**
You send it latitude/longitude points, and it gives you simple ways to:

* Save positions.
* Read back the latest position of an object.
* Query all points inside a rectangular area.
* Delete points.

It’s written in Go, optimized for high throughput, and designed to run as a **small cluster** of nodes (e.g. on Kubernetes). Nodes discover each other via a **gossip-based membership system** and keep state **best-effort synchronized** across the cluster.

If you’re building anything that keeps track of “things on a map” and needs to read/write them quickly, Loggerhead is meant to be the geospatial engine you don’t have to think about.

---

## Why Loggerhead?

**Straightforward mental model**

* Store points as `(namespace, id, lat, lon)`.
* Query by **ID** (`GET`) or by **area** (`POLY`).
* Use a simple text protocol over TCP (`SAVE`, `GET`, `DELETE`, `POLY`).

**Fast in-memory engine**

* Geospatial data is kept in memory and indexed with a **quadtree**.
* Benchmarks (on an AMD EPYC 7763) show:

  * ~20–25M `GetLocation` lookups per second.
  * ~500k `Save` operations per second.
  * City-scale radius queries in ~200 µs, dropping under 50 µs on 4 cores.
  * Continent-scale radius queries in ~10–15 ms on 4 cores.

**Cluster-aware**

* Nodes use **gossip** to discover each other and share state.
* Best-effort synchronization between nodes.
* Works nicely with DNS-based discovery in Kubernetes.

**Operational hooks**

* **Prometheus metrics** exposed over HTTP.
* Basic **admin interface** to visualize cluster state.

Loggerhead is intentionally focused: a fast, in-memory geospatial store with a small surface area. You can pair it with your existing databases and services without changing your whole stack.

---

## Quick Start

### Build

Loggerhead requires **Go 1.22.1** and **GCC** to build.

```bash
CGO_ENABLED=1 GOARCH=$TARGETARCH go build -o loggerhead
```

### Run a node

```bash
./loggerhead --cluster-dns=loggerhead.default.svc.cluster.local
```

Sample output:

```text
2024/06/10 01:44:07 Please set the following environment variables:
2024/06/10 01:44:07 CLUSTER_DNS
2024/06/10 01:44:07 Reverting to flags...
2024/06/10 01:44:07 [DEBUG] memberlist: Initiating push/pull sync with:  [::1]:20001
2024/06/10 01:44:07 [DEBUG] memberlist: Stream connection from=[::1]:42194
2024/06/10 01:44:07 Sharing local state to a new node
...
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

### Save and read your first point

Open a terminal:

```bash
telnet localhost 19999
```

Save a point:

```text
SAVE mynamespace myid 12.560000 13.560000
>> 1.0,saved
```

Read it back:

```bash
telnet localhost 19998
```

```text
GET mynamespace myid
>> 1.0,mynamespace,myid,12.560000,13.560000
```

Query everything in an area (POLY):

```text
POLY mynamespace 10.560000 10.560000 15.560000 15.560000
>> 1.0,mynamespace,myid,12.560000,13.560000
>> 1.0,mynamespace,myid2,12.560000,11.560000
>> 1.0,mynamespace,myid3,14.560000,13.560000
>> 1.0,done
```

> Note: the `1.0` prefix is the **protocol version**, so clients can detect changes in the future.

---

## Ports & Architecture

A running Loggerhead node exposes several ports:

* **19998** – Read queries (`GET`, `POLY`).
* **19999** – Write queries (`SAVE`, `DELETE`).
* **20000** – HTTP admin interface & `/metrics` endpoint (Prometheus).
* **20001** – Gossip port for cluster communication.

You typically run **multiple nodes**, point them at the same `CLUSTER_DNS`, and let Loggerhead handle discovery and membership via gossip.

---

## Configuration

You can configure Loggerhead using **environment variables** or **command-line flags**.

Currently supported:

* **`CLUSTER_DNS`**
  DNS name used to discover other nodes.
  Loggerhead looks up this DNS record and uses the IPs as peers.

  This is particularly convenient in Kubernetes; you can provide the service DNS (for example `loggerhead.default.svc.cluster.local`), and nodes will discover each other. When you scale up, new nodes automatically join the cluster.

* **`MAX_CONNECTIONS`**
  Maximum number of connections allowed per port (separately for READ and WRITE).
  Too few connections can create congestion per CPU core; too many can push CPU to 100% and slow everything down. Loggerhead is usually called by backend services, so you rarely need to expose huge numbers of connections.
  If you need many connections, you may also have to adjust `ulimit` on Linux.

* **`SEED_NODES`** *(coming soon)*
  Planned: a list of seed nodes to bootstrap the cluster.

---

## Query Language

Loggerhead speaks a very small text protocol over TCP. Each message starts with a version prefix (`1.0` currently).

### Reading (port 19998)

#### GET

Get the last known position for a given `(namespace, id)`:

```text
telnet localhost 19998
GET mynamespace myid

>> 1.0,mynamespace,myid,12.560000,13.560000
```

#### POLY

Get all points in a rectangular area (min lat/lon, max lat/lon):

```text
telnet localhost 19998
POLY mynamespace 10.560000 10.560000 15.560000 15.560000
>> 1.0,mynamespace,myid,12.560000,13.560000
>> 1.0,mynamespace,myid2,12.560000,11.560000
>> 1.0,mynamespace,myid3,14.560000,13.560000
>> 1.0,done
```

### Writing (port 19999)

> Tip: use short names for `namespace` and `id` when possible. Loggerhead uses Go maps internally, and shorter string keys can be slightly faster.

#### SAVE

Insert or update a point:

```text
telnet localhost 19999
SAVE mynamespace myid 12.560000 13.560000
>> 1.0,saved
```

#### DELETE

Remove a point:

```text
telnet localhost 19999
DELETE mynamespace myid
>> 1.0,deleted
```

---

## Performance

The in-memory engine has been benchmarked on an **AMD EPYC 7763 64-core processor** using Go 1.22.1.

Headline numbers (for 1–4 cores):

* ~20–25M `GetLocation` lookups per second.
* ~500k `Save` operations per second.
* City-scale radius queries in ~200 µs, dropping under 50 µs on 4 cores.
* Continent-scale radius queries in ~10–15 ms on 4 cores.

**Delete performance note**

Deletes are currently protected by a **global/index-level lock**. Under synthetic benchmarks that hammer deletes on 4 cores, this shows up as contention and increased latency. In most real workloads, deletes are rare compared to reads and writes, but this is a known area to optimize.

---

## Benchmark of the Core World Engine

These benchmarks test the **core in-memory engine** only. They do **not** include network or protocol overhead, to keep the numbers comparable across environments.

* Benchmark duration: **2 seconds** per run.
* Cores tested: **1, 2, 4, 8, 16, 32** (only 1 / 2 / 4 shown here).
* Expect a slight decrease in end-to-end performance when using a real network.

### Engine running on 1 core

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 2,691,468      | 1,982          | 216          | 1           |
| GetLocation          | Return a single location              | 54,935,690     | 41.60          | 0            | 0           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 10,000         | 212,897        | 85,952       | 83          |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 100            | 82,167,521     | 26,660,673   | 11,594      |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 100            | 99,728,196     | 32,027,345   | 13,281      |
| Delete               | Delete a location                     | 54,722,920     | 46.31          | 7            | 0           |

### Engine running on 2 cores

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 3,910,928      | 1,129          | 130          | 1           |
| GetLocation          | Return a single location              | 28,343,620     | 84.74          | 0            | 0           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 37,132         | 77,616         | 58,016       | 78          |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 100            | 23,398,650     | 18,263,379   | 10,474      |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 100            | 30,380,619     | 21,837,460   | 12,006      |
| Delete               | Delete a location                     | 48,852,799     | 50.14          | 7            | 0           |

### Engine running on 4 cores

| Operation            | Scenario / Description                | Iterations (N) | Time / op (ns) | Mem / op (B) | Allocs / op |
| -------------------- | ------------------------------------- | -------------- | -------------- | ------------ | ----------- |
| Save                 | Save a new location                   | 5,615,270      | 671.8          | 62           | 1           |
| GetLocation          | Return a single location              | 48,844,772     | 49.58          | 0            | 0           |
| GetLocationsInRadius | Locations in the UAE (~83.6k km²)     | 69,660         | 43,884         | 40,288       | 44          |
| GetLocationsInRadius | Locations in the USA (~9.8M km²)      | 184            | 11,285,953     | 12,295,469   | 9,142       |
| GetLocationsInRadius | Locations in all of Africa (~30M km²) | 171            | 14,131,393     | 15,556,630   | 10,700      |
| Delete               | Delete a location                     | 6,145,867      | 328.1          | 7            | 0           |

---

## Roadmap to 0.1.0

Loggerhead is still early, but there’s a clear path for where it’s going.

### Planned

* [ ] **Polygon subscriptions** – subscribe to updates for a polygon and receive changes. Planned for `0.0.4`.
* [ ] **Realistic benchmarks with ADS-B traffic** – use about a week of global ADS-B data for stress-testing. Planned for `0.0.5`.
* [ ] **Optional RAFT-based consistency** – enable a RAFT mode for stronger consistency within a cluster (trading some performance for guarantees). Planned for `0.1.0`.
* [ ] **Sharding by namespace** – shard namespaces across TreeNodes with primary + replication (multiple RAFT groups in parallel). Planned for `0.2.0`.
* [ ] **Durability** – store and recover state from disk. Planned for `0.3.0`.

### Already done

* [x] Improve Prometheus metrics (`0.0.3`).
* [x] Reduce clustering chatter to avoid saturating the network (`0.0.2`).
* [x] Connect query language to the database.
* [x] Wire network interface to the query processor.
* [x] Implement in-memory storage using a **quadtree**.
* [x] Implement storage benchmarks.
* [x] Implement network interface.
* [x] Implement query language.
* [x] Implement clustering.
* [x] Implement Prometheus metrics.
* [x] Implement admin interface.
* [x] Implement gossip protocol.

---

If you have ideas, issues, or a workload you’d like to try on Loggerhead, opening an issue or sharing your use case will directly shape where this engine goes next.
