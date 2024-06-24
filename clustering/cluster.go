package clustering

import (
	"errors"
	"github.com/fabricekabongo/loggerhead/config"
	"github.com/fabricekabongo/loggerhead/query"
	"github.com/hashicorp/memberlist"
	"log"
	"net"
	"os"
	"time"
)

var (
	FailedToJoinCluster       = errors.New("failed to join cluster")
	FailedToCreateCluster     = errors.New("failed to create cluster")
	FailedToExtractIPsFromDNS = errors.New("failed to extract IPs from DNS")
)

type Cluster struct {
	memberList *memberlist.Memberlist
	broadcasts *memberlist.TransmitLimitedQueue
}

func (c *Cluster) Close(timeout time.Duration) error {
	err := c.memberList.Leave(timeout)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cluster) Broadcasts() *memberlist.TransmitLimitedQueue {
	return c.broadcasts
}

func (c *Cluster) MemberList() *memberlist.Memberlist {
	return c.memberList
}

func NewCluster(engine *query.Engine, config config.Config) (*Cluster, error) {
	broadcasts := &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return 1 // Replace with the actual number of nodes
		},
		RetransmitMult: 3,
	}

	delegate := newBroadcastDelegate(engine, broadcasts)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal("Failed to get hostname: ", err)
	}

	cfg := memberlist.DefaultLocalConfig()
	cfg.Name = hostname
	cfg.BindPort = config.ClusterPort
	cfg.AdvertisePort = config.ClusterPort
	cfg.Delegate = delegate

	mList, err := memberlist.Create(cfg)
	if err != nil {
		log.Println("Failed to create cluster: ", err)
		return nil, FailedToCreateCluster
	}

	cluster := &Cluster{
		memberList: mList,
		broadcasts: broadcasts,
	}

	broadcasts.NumNodes = func() int {
		return mList.NumMembers()
	}

	clusterIPs, err := getClusterIPs(config)
	if err != nil {
		return cluster, FailedToExtractIPsFromDNS
	}

	_, err = mList.Join(clusterIPs)
	if err != nil {
		return cluster, FailedToJoinCluster
	}

	return cluster, nil
}

func getClusterIPs(config config.Config) ([]string, error) {
	var clusterIPs []string

	if config.SeedNode != "" {
		clusterIPs = []string{config.SeedNode}
	} else {
		ips, err := getIPsFromDomainName(config.ClusterDNS)
		if err != nil {
			return nil, FailedToExtractIPsFromDNS
		}

		clusterIPs = ips
	}

	return clusterIPs, nil
}

func getIPsFromDomainName(clusterDNS string) ([]string, error) {
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
