package config

import (
	"fmt"
	"sync/atomic"
)

// Supported ONLY bool,int32|64,string,time.Duration
type DynamicFlag[T comparable] struct {
	v *atomic.Value
}

func NewDynamicFlag[T comparable](v T) *DynamicFlag[T] {
	var av atomic.Value
	av.Store(v)

	return &DynamicFlag[T]{
		v: &av,
	}
}

func (m *DynamicFlag[T]) Store(v *T) error {
	m.v.Store(*v)
	return nil
}

func (m *DynamicFlag[T]) Load() *T {
	v := m.v.Load().(T)
	return &v
}

//
// urfave cli.Generic interface compatibility methods

// fake method: do nothing
func (m *DynamicFlag[T]) Set(string) error {
	return nil
}

// fake method: output human value (for --help)
func (m *DynamicFlag[T]) String() string {
	return fmt.Sprint(m.v.Load())
}
