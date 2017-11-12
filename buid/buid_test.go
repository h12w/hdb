package buid

import (
	"testing"
	"time"
)

func BenchmarkNewID(b *testing.B) {
	node := NewProcess(1)
	t := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		id := node.NewID(2, t)
		_ = id
		t = t.Add(time.Nanosecond)
	}
}
