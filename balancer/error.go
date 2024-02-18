package balancer

import (
	"fmt"
)

const (
	errUndefined = "undefined error occurred"

	errUnparsableChunk = "could not get key from given chunkname %s"
	errLockMiss        = "could not get balancer read-lock"

	errUpstreamIsDown = "balancer is marked as down"
	// errUpstreamIsEmpty = "balancer's upstream has no servers"
	errServerIsDown = "server %s is marked as down"
)

type BalancerEFlag uint8

// IsRetriable - can be retried in current upstream
// IsNextServerRouted (IsBackupable) - must be routed to the next cluster's server
// IsNextClusterRouted (IsReroutable) - must be routed to the next cluster
const (
	IsRetriable BalancerEFlag = 1 << iota
	IsBackupable
	IsReroutable
)

type BalancerError struct {
	Balancer Balancer
	Err      string

	flags BalancerEFlag
}

func NewError(m Balancer, e string) (b *BalancerError) {
	b = &BalancerError{Balancer: m}
	return b.NewError(e)
}

func NewErrorF(m Balancer, format string, args ...any) (b *BalancerError) {
	b = &BalancerError{Balancer: m}
	return b.NewErrorF(format, args...)
}

func (m *BalancerError) NewError(e string) *BalancerError {
	m.Err = e
	return m
}

func (m *BalancerError) NewErrorF(format string, args ...any) *BalancerError {
	m.Err = fmt.Sprintf(format+"\n", args...)
	return m
}

func (m *BalancerError) HasError() bool {
	return len(m.Err) != 0
}

func (m *BalancerError) Error() string {
	return fmt.Sprintf("cluster %s failed with %s", m.Balancer.GetClusterName(), m.Err)
}

func (m *BalancerError) SetFlag(flag BalancerEFlag) *BalancerError {
	m.flags = m.flags ^ flag
	return m
}

func (m *BalancerError) Has(flag BalancerEFlag) bool {
	return m.flags&flag != 0
}
