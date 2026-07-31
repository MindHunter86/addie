package config

import (
	"strconv"
	"time"

	"go.uber.org/atomic"
)

type DynamicString struct {
	s atomic.String
}

func NewDynamicString(s string) *DynamicString {
	return &DynamicString{
		s: *atomic.NewString(s),
	}
}

func (m *DynamicString) Load() string {
	return m.s.Load()
}

func (m *DynamicString) String() string {
	return m.String()
}

func (m *DynamicString) Set(s string) error {
	m.s.Store(s)
	return nil
}

// Supported ONLY bool,int32|64,string,time.Duration
type DynamicFlag[T comparable] struct {
	v atomic.Value
}

func (m *DynamicFlag[T]) Set(v T) error {
	m.v.Store(v)
	return nil
}

func (m *DynamicFlag[T]) Get() *T {
	v := m.v.Load().(T)
	return &v
}

func (m *DynamicFlag[T]) String() string {
	switch v := m.v.Load().(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(int(v))
	case int64:
		return strconv.Itoa(int(v))
	case string:
		return v
	case time.Duration:
		return v.String()
	default:
		return ""
	}
}
