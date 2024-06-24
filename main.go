package main

import (
	"fmt"
	"github.com/fabricekabongo/loggerhead/clustering"
	"github.com/fabricekabongo/loggerhead/config"
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/server"
	"github.com/fabricekabongo/loggerhead/world"
	"log"
	"time"
)

func main() {
	start := time.Now()
	cfg := config.GetConfig()

	worldMap := world.NewWorld()
	readEngine := query.NewReadQueryEngine(worldMap)
	writeEngine := query.NewWriteQueryEngine(worldMap)

	cluster, err := clustering.NewCluster(writeEngine, cfg)

	if err != nil {
		log.Fatal("Failed to create cluster: ", err)
	}

	defer func(cluster *clustering.Cluster) {
		err := cluster.Close(0)
		if err != nil {
			log.Println("Failed to leave cluster: ", err)
		}
	}(cluster)
	//
	//opsServer := admin.NewOpsServer(cluster, cfg)
	//go opsServer.Start()

	writer := server.NewListener(cfg, writeEngine)
	reader := server.NewListener(cfg, readEngine)

	svr := server.NewServer([]*server.Listener{writer, reader})

	end := time.Now()
	fmt.Println("Startup time: ", end.Sub(start))

	defer svr.Stop()

	printWelcomeMessage(cfg, cluster)

	svr.Start()
}

func printWelcomeMessage(cfg config.Config, cluster *clustering.Cluster) {
	fmt.Println("===========================================================")
	fmt.Println("Starting the Database Server")
	fmt.Println("===========================================================")
	fmt.Println("Read Port: ", cfg.ReadPort)
	fmt.Println("Write Port: ", cfg.WritePort)
	fmt.Println("Cluster Port: ", cfg.ClusterPort)
	fmt.Println("Max Connections: ", cfg.MaxConnections)
	fmt.Println("Max EOF Wait: ", cfg.MaxEOFWait)
	fmt.Println("Cluster DNS: ", cfg.ClusterDNS)
	fmt.Println("Seed Node: ", cfg.SeedNode)
	fmt.Println("My IP: ", cluster.MemberList().LocalNode().Addr.String())
	fmt.Println("Node Name: ", cluster.MemberList().LocalNode().Name)
	fmt.Println("Node State: ", cluster.MemberList().LocalNode().State)
	fmt.Println("===========================================================")
}
