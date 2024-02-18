package balancer

import (
	"io"
	"net"
)

type Balancer interface {
	BalanceByChunkname(prefix, chunkname string, try uint8) (_ string, server *BalancerServer, e error)
	UpdateServers(servers map[string]net.IP)
	GetStats() io.Reader
	ResetStats()
	ResetUpstream()
	GetClusterName() string
}

type BalancerCluster uint8

const (
	BalancerClusterNodes BalancerCluster = iota
	BalancerClusterCloud
)

var GetBalancerByString = map[string]BalancerCluster{
	"cache-nodes": BalancerClusterNodes,
	"cache-cloud": BalancerClusterCloud,
}

var MaxTries = uint8(3)
