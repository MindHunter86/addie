package utils

import "context"

func ContextValueExtract[V comparable](c context.Context, k ContextKey) V {
	v := c.Value(k).(V)
	return v
}
