package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/fabricekabongo/loggerhead/admin"
	"github.com/fabricekabongo/loggerhead/clustering"
	"github.com/fabricekabongo/loggerhead/query"
	server2 "github.com/fabricekabongo/loggerhead/server"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	FailedToJoinCluster       = errors.New("failed to join cluster")
	FailedToCreateCluster     = errors.New("failed to create cluster")
	FailedToExtractIPsFromDNS = errors.New("failed to extract IPs from DNS")
)

func main() {
	start := time.Now()
	clusterDNS, maxConnections, seedNode := populateEnv()

	worldMap := world.NewWorld()

	mList, broadcasts, err := createClustering(clusterDNS, worldMap, seedNode)

	if errors.Is(err, FailedToCreateCluster) {
		log.Fatal("Failed to create cluster: ", err)
	}
	if errors.Is(err, FailedToJoinCluster) {
		log.Println("Failed to join cluster: ", err)
	}
	if errors.Is(err, FailedToExtractIPsFromDNS) {
		log.Println("Failed to extract IPs from DNS: ", err)
	}

	defer func(mList *memberlist.Memberlist, timeout time.Duration) {
		err := mList.Leave(timeout)
		if err != nil {
			log.Println("Failed to leave cluster: ", err)
		}
	}(mList, 0)

	fmt.Println("===========================================================")
	fmt.Println("Starting the Database Server")
	fmt.Println("Cluster DNS: ", clusterDNS)
	fmt.Println("Use the following ports for the following services:")
	fmt.Println("Reading location update: 19998")
	fmt.Println("Writing location update: 19999")
	fmt.Println("Max Concurrent Connections per port (19999 & 19998):", maxConnections)
	fmt.Println("Admin UI (/) & Metrics(/metrics): 20000")
	fmt.Println("Clustering: 20001")
	fmt.Println("My IP is: ", mList.LocalNode().Addr.String())
	fmt.Println("===========================================================")
	opsServer := admin.NewOpsServer(mList, worldMap)
	go opsServer.Start()

	readEngine := query.NewReadQueryEngine(worldMap)
	writeEngine := query.NewWriteQueryEngine(worldMap)

	writer := server2.NewWriteHandler(writeEngine, broadcasts, maxConnections)
	reader := server2.NewReadHandler(readEngine, maxConnections)

	server := server2.NewServer(*writer, *reader)

	end := time.Now()
	fmt.Println("Startup time: ", end.Sub(start))

	defer server.Stop()
	server.Start()
}

func createClustering(clusterDNS string, world *world.World, seedNode string) (*memberlist.Memberlist, *memberlist.TransmitLimitedQueue, error) {
	broadcasts := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return 1 // Replace with the actual number of nodes
		},
		RetransmitMult: 3,
	}

	delegate := clustering.NewBroadcastDelegate(world, broadcasts)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal("Failed to get hostname: ", err)
	}

	config := memberlist.DefaultLocalConfig()
	config.Name = hostname
	config.BindPort = 20001
	config.AdvertisePort = 20001
	config.Delegate = delegate

	mList, err := memberlist.Create(config)
	if err != nil {
		log.Println("Failed to create cluster: ", err)
		return nil, nil, FailedToCreateCluster
	}

	broadcasts.NumNodes = func() int {
		return mList.NumMembers()
	}
	var clusterIPs []string

	if seedNode != "" {
		clusterIPs = []string{seedNode}
	} else {
		clusterIPs, err = getClusterIPs(clusterDNS)
		if err != nil {
			log.Println("Failed to get cluster IPs: ", err)
			log.Println("Cluster DNS: ", clusterDNS)
			log.Println("I assume I am the seed node. I will work alone until the other nodes join me.")

			return mList, broadcasts, nil
		}
	}

	_, err = mList.Join(clusterIPs)
	if err != nil {
		log.Println("Failed to join cluster: ", err)
		return mList, broadcasts, FailedToJoinCluster
	}

	return mList, broadcasts, nil
}

func getClusterIPs(clusterDNS string) ([]string, error) {
	ips, err := net.LookupIP(clusterDNS)
	if err != nil {
		return nil, err
	}
	currentIp, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	// map addresses to strings
	var clusterIPs []string
	for _, ip := range ips {
		if ip.String() == currentIp {
			continue
		}
		clusterIPs = append(clusterIPs, ip.String())
	}

	return clusterIPs, nil
}

func populateEnv() (string, int, string) {

	envClusterDNS := os.Getenv("CLUSTER_DNS")
	envMaxConnections := os.Getenv("MAX_CONNECTIONS")
	envSeedNode := os.Getenv("SEED_NODE")

	var flagClusterDNS string
	var flagMaxConnections int
	var flagSeedNode string

	var clusterDNS string
	var seedNode string
	var maxConnections int

	flag.StringVar(&flagClusterDNS, "cluster-dns", "", "Cluster DNS")
	flag.StringVar(&flagSeedNode, "seed-node", "", "Seed Node IP Address")
	flag.IntVar(&flagMaxConnections, "max-connections", 20, "Max connections concurrently per port (eg: 20 read, 20 write). Default: 20. Remember this database is supposed to be called by your backend services not by your consumers. So you shouldn't need too many connections.")
	flag.Parse()

	if envClusterDNS == "" {
		clusterDNS = flagClusterDNS

		if clusterDNS == "" {
			log.Println("No environment variable set for CLUSTER_DNS or flag set for cluster-dns")
		}
	} else {
		clusterDNS = envClusterDNS
	}

	if envSeedNode == "" {
		seedNode = flagSeedNode

		if seedNode == "" {
			log.Println("No environment variable set for SEED_NODE or flag set for seed-node")
		}
	} else {
		seedNode = envSeedNode
	}

	if envMaxConnections == "" {
		if flagMaxConnections < 1 {
			log.Fatalln("Max connections should be greater than 0")
		}

		maxConnections = flagMaxConnections

	} else {
		convMaxConnections, err := strconv.Atoi(envMaxConnections)
		if err != nil {
			log.Fatalln("Failed to convert max connections to int")
		}

		if convMaxConnections < 1 {
			log.Fatalln("Max connections should be greater than 0")
		}

		maxConnections = convMaxConnections
	}

	return clusterDNS, maxConnections, seedNode
}
