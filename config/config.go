package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

var (
	envClusterDNS  = os.Getenv("CLUSTER_DNS")
	flagClusterDNS string

	envMaxConnections, envMaxErr = strconv.Atoi(os.Getenv("MAX_CONNECTIONS"))
	flagMaxConnections           int

	envSeedNode  = os.Getenv("SEED_NODE")
	flagSeedNode string

	envReadPort, envReadPortErr = strconv.Atoi(os.Getenv("READ_PORT"))
	flagReadPort                int

	envWritePort, envWritePortErr = strconv.Atoi(os.Getenv("WRITE_PORT"))
	flagWritePort                 int

	envSubPort, envSubPortErr = strconv.Atoi(os.Getenv("SUB_PORT"))
	flagSubPort               int

	envHttpPort, envHttpPortErr = strconv.Atoi(os.Getenv("HTTP_PORT"))
	flagHttpPort                int

	envClusterPort, envClusterPortErr = strconv.Atoi(os.Getenv("CLUSTER_PORT"))
	flagClusterPort                   int

	envMaxEOFWait, envMaxEOFWaitErr = strconv.Atoi(os.Getenv("MAX_EOF_WAIT"))
	flagMaxEOFWait                  int
)

type Config struct {
	ClusterDNS     string
	MaxConnections int
	SeedNode       string
	ReadPort       int
	WritePort      int
	SubPort        int
	HttpPort       int
	ClusterPort    int
	MaxEOFWait     time.Duration
}

func parseFlags() {
	flag.StringVar(&flagClusterDNS, "cluster-dns", "", "Cluster DNS")
	flag.StringVar(&flagSeedNode, "seed-node", "", "Seed Node IP Address")
	flag.IntVar(&flagMaxConnections, "max-connections", 20, "Max connections concurrently per port (eg: 20 read, 20 write). Default: 20. Remember this database is supposed to be called by your backend services not by your consumers. So you shouldn't need too many connections.")
	flag.IntVar(&flagReadPort, "read-port", 19998, "Read port. Default: 19998")
	flag.IntVar(&flagWritePort, "write-port", 19999, "Write port. Default: 19999")
	flag.IntVar(&flagSubPort, "sub-port", 20001, "Subscription port. Default: 20001")
	flag.IntVar(&flagHttpPort, "http-port", 20000, "HTTP port. Default: 20000")
	flag.IntVar(&flagClusterPort, "cluster-port", 20001, "Cluster port. Default: 20001")
	flag.IntVar(&flagMaxEOFWait, "max-eof-wait", 30, "Max EOF wait time in seconds. Default: 30")

	flag.Parse()
}

func GetConfig() Config {
	parseFlags()

	return Config{
		ClusterDNS:     processClusterDNS(),
		MaxConnections: processMaxConnections(),
		SeedNode:       processSeedNode(),
		ReadPort:       processReadPort(),
		WritePort:      processWritePort(),
		SubPort:        processSubPort(),
		HttpPort:       processHttpPort(),
		ClusterPort:    processClusterPort(),
		MaxEOFWait:     processMaxEOFWait(),
	}
}

func processClusterDNS() string {
	if flagClusterDNS != "" {
		return flagClusterDNS
	}
	if envClusterDNS != "" {
		return envClusterDNS
	}
	return ""
}

func processMaxEOFWait() time.Duration {
	if envMaxEOFWaitErr == nil && envMaxEOFWait > 0 {
		return time.Duration(envMaxEOFWait) * time.Second
	}
	return time.Duration(flagMaxEOFWait) * time.Second
}

func processMaxConnections() int {
	if envMaxErr == nil && envMaxConnections > 0 {
		return envMaxConnections
	}
	return flagMaxConnections
}

func processSeedNode() string {
	if flagSeedNode != "" {
		return flagSeedNode
	}
	if envSeedNode != "" {
		return envSeedNode
	}
	return ""
}

func processReadPort() int {
	if envReadPortErr == nil && envReadPort > 0 {
		return envReadPort
	}
	return flagReadPort
}

func processWritePort() int {
	if envWritePortErr == nil && envWritePort > 0 {
		return envWritePort
	}
	return flagWritePort
}

func processSubPort() int {
	if envSubPortErr == nil && envSubPort > 0 {
		return envSubPort
	}
	return flagSubPort
}

func processHttpPort() int {
	if envHttpPortErr == nil && envHttpPort > 0 {
		return envHttpPort
	}
	return flagHttpPort
}

func processClusterPort() int {
	if envClusterPortErr == nil && envClusterPort > 0 {
		return envClusterPort
	}
	return flagClusterPort
}
