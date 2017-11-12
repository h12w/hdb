package buid

import "testing"

func BenchmarkNewID(b *testing.B) {
	node := NewNode(1)

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		id := node.NewID(2)
		_ = id
	}
}
