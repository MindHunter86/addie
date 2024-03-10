package balancer

import (
	"bytes"
	"errors"
	"io"
	"net"
)

type Balancer interface {
	BalanceByChunk(prefixbuf *bytes.Buffer, chunkname []byte) (_ string, server *BalancerServer, e error)
	BalanceRandom() (_ string, server *BalancerServer, e error)
	UpdateServers(servers map[string]net.IP)
	GetStats() io.Reader
	ResetStats()
	ResetUpstream()
	GetClusterName() string
}

var (
	ErrUnparsableChunk     = errors.New("could not get server because of invalid chunk name")
	ErrServerUnavailable   = errors.New("rolled server is down now")
	ErrUpstreamUnavailable = errors.New("upstream is empty or undefined; balancing is not possible")
)

type BalancerCluster uint8

const (
	BalancerClusterNodes BalancerCluster = iota
	BalancerClusterCloud
)

var GetBalancerByString = map[string]BalancerCluster{
	"cache-nodes": BalancerClusterNodes,
	"cache-cloud": BalancerClusterCloud,
}
