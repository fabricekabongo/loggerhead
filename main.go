package main

import (
	"errors"
	"fmt"
	"github.com/fabricekabongo/loggerhead/admin"
	"github.com/fabricekabongo/loggerhead/clustering"
	"github.com/fabricekabongo/loggerhead/config"
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/fabricekabongo/loggerhead/server"
	"github.com/fabricekabongo/loggerhead/world"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
)

func main() {
	f, err := os.Create("loggerhead.prof")
	if err != nil {
		panic("Failed to create profile file")
	}

	err = pprof.StartCPUProfile(f)
	if err != nil {
		log.Fatal("Failed to start CPU profile: ", err)
	}

	defer pprof.StopCPUProfile()
	start := time.Now()
	cfg := config.GetConfig()

	worldMap := world.NewWorld()
	readEngine := query.NewReadQueryEngine(worldMap)
	writeEngine := query.NewWriteQueryEngine(worldMap)
	// subscriberEngine := query.NewSubscriberQueryEngine(worldMap)

	cluster, err := clustering.NewCluster(writeEngine, cfg)

	if err != nil {
		if errors.Is(err, clustering.FailedToCreateCluster) {
			log.Fatal("Failed to create cluster: ", err)
		} else {
			log.Println(err)
		}
	}

	defer func(cluster *clustering.Cluster) {
		err := cluster.Close(0)
		if err != nil {
			log.Println("Failed to leave cluster: ", err)
		}
	}(cluster)

	clusterEngine := clustering.NewEngineDecorator(cluster, writeEngine)

	opsServer := admin.NewOpsServer(cluster, cfg)
	go opsServer.Start()

	writer := server.NewListener(cfg.WritePort, cfg.MaxConnections, cfg.MaxEOFWait, clusterEngine) // This is the writer listener (for writes and broadcasts)
	reader := server.NewListener(cfg.ReadPort, cfg.MaxConnections, cfg.MaxEOFWait, readEngine)     // This is the reader listener (for reads).
	// subscriber := server.NewListener(cfg, subscriberEngine)

	svr := server.NewServer([]*server.Listener{writer, reader})

	end := time.Now()
	fmt.Println("Startup time: ", end.Sub(start))

	defer svr.Stop()

	printWelcomeMessage(cfg, cluster)
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		s := <-sigc
		log.Println("Received signal: ", s)
		svr.Stop()
		close(sigc)
		err := cluster.Close(0)
		if err != nil {
			return
		}
		os.Exit(0)
	}()

	svr.Start()

	return
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
	fmt.Println("Node State: ", clustering.StateToString(cluster.MemberList().LocalNode().State))
	fmt.Println("===========================================================")
}
