package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fabricekabongo/loggerhead/admin"
	"github.com/fabricekabongo/loggerhead/clustering"
	"github.com/fabricekabongo/loggerhead/config"
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/server"
	"github.com/fabricekabongo/loggerhead/world"
)

func main() {

	ctx := context.Background()
	start := time.Now()
	cfg := config.GetConfig()

	worldMap := world.NewWorld()
	readEngine := query.NewReadQueryEngine(worldMap)
	writeEngine := query.NewWriteQueryEngine(worldMap)
	// subscriberEngine := query.NewSubscriberQueryEngine(worldMap)

	cluster, err := clustering.NewCluster(writeEngine, cfg)

	if err != nil {
		log.Println(err)

		if errors.Is(err, clustering.ErrFailedToCreateCluster) {
			log.Fatal("Failed to create cluster: ", err)
		}
	}

	defer func(cluster *clustering.Cluster) {
		err := cluster.Close(0)
		if err != nil {
			log.Println("Failed to leave cluster: ", err)
		}
	}(cluster)

	ClusterCtx, concel := context.WithCancel(ctx)
	clusterEngine := clustering.NewEngineDecorator(ClusterCtx, cluster, writeEngine)

	opsServer := admin.NewOpsServer(cluster, cfg)
	go opsServer.Start()

	writer := server.NewListener(cfg.WritePort, cfg.MaxConnections, cfg.MaxEOFWait, clusterEngine) // This is the writer listener (for writes and broadcasts)
	reader := server.NewListener(cfg.ReadPort, cfg.MaxConnections, cfg.MaxEOFWait, readEngine)     // This is the reader listener (for reads).
	// subscriber := server.NewListener(cfg, subscriberEngine)

	svr := server.NewServer([]*server.Listener{writer, reader})

	defer svr.Stop()

	printWelcomeMessage(cfg, cluster)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func(cancel context.CancelFunc) {
		s := <-sigc
		log.Println("Received signal: ", s)
		svr.Stop()
		close(sigc)
		cancel()

		err := cluster.Close(0)
		if err != nil {
			return
		}
		os.Exit(0)
	}(concel)

	end := time.Now()
	fmt.Println("Startup time: ", end.Sub(start))
	svr.Start()
}

func printWelcomeMessage(cfg config.Config, cluster *clustering.Cluster) {
	fmt.Println("===========================================================")
	fmt.Println("Starting the Database Server")
	fmt.Println("===========================================================")
	fmt.Println("Read Port: ", cfg.ReadPort)
	fmt.Println("Write Port: ", cfg.WritePort)
	fmt.Println("Cluster Port: ", cfg.ClusterPort)
	fmt.Println("Admin & Prometheus Port:", cfg.HttpPort)
	fmt.Println("Max Connections: ", cfg.MaxConnections)
	fmt.Println("Max EOF Wait: ", cfg.MaxEOFWait)
	fmt.Println("Cluster DNS: ", cfg.ClusterDNS)
	fmt.Println("Seed Node: ", cfg.SeedNode)
	fmt.Println("My IP: ", cluster.MemberList().LocalNode().Addr.String())
	fmt.Println("Node Name: ", cluster.MemberList().LocalNode().Name)
	fmt.Println("Node State: ", clustering.StateToString(cluster.MemberList().LocalNode().State))
	fmt.Println("===========================================================")
}
