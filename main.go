package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"github.com/fabricekabongo/loggerhead/admin"
	"github.com/fabricekabongo/loggerhead/clustering"
	server2 "github.com/fabricekabongo/loggerhead/server"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
	"log"
	"net"
	"os"
	"time"
)

var (
	clusterDNS                = os.Getenv("CLUSTER_DNS")
	FailedToJoinCluster       = errors.New("failed to join cluster")
	FailedToCreateCluster     = errors.New("failed to create cluster")
	FailedToExtractIPsFromDNS = errors.New("failed to extract IPs from DNS")
)

func main() {

	populateEnv()

	flag.Parse()

	gob.Register(&world.Location{})
	gob.Register(&world.Grid{})
	gob.Register(&world.Map{})

	worldMap := world.NewMap()

	mList, broadcasts, err := createClustering(clusterDNS, worldMap)
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
	fmt.Println("Writing location update: 19999")
	fmt.Println("Reading location update: 19998")
	fmt.Println("Admin UI (/) & Metrics(/metrics): 20000")
	fmt.Println("Clustering: 20001")
	fmt.Println("===========================================================")
	opsServer := admin.NewOpsServer(mList, worldMap)
	go opsServer.Start()

	writer := server2.NewWriteHandler(worldMap, broadcasts)
	reader := server2.NewReadHandler(worldMap)

	server := server2.NewServer(*writer, *reader)

	server.Start()
}

func createClustering(clusterDNS string, world *world.Map) (*memberlist.Memberlist, *memberlist.TransmitLimitedQueue, error) {
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

	if err != nil {
		log.Println("Failed to get hostname: ", err)
	}

	mList, err := memberlist.Create(config)
	if err != nil {
		log.Println("Failed to create cluster: ", err)
		return nil, nil, FailedToCreateCluster
	}

	broadcasts.NumNodes = func() int {
		return mList.NumMembers()
	}

	clusterIPs, err := getClusterIPs(clusterDNS)
	if err != nil {
		log.Println("Failed to get cluster IPs: ", err)
		return nil, nil, FailedToExtractIPsFromDNS
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

func populateEnv() {
	if clusterDNS == "" {
		log.Println("Please set the following environment variables:")
		log.Println("CLUSTER_DNS")
		log.Println("Reverting to flags...")

		flag.StringVar(&clusterDNS, "cluster-dns", "", "Cluster DNS")
		flag.Parse()

		if clusterDNS == "" {
			log.Println("No flags set. Please set the following flags:")
			log.Println("cluster-dns")
			os.Exit(1)
		}
	}
}
