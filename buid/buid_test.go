package buid

import (
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	process := NewProcess(1)
	ts := time.Now().UTC().Truncate(time.Microsecond)
	id := process.NewID(2, ts)
	if extractedTs := id.Time(); !extractedTs.Equal(ts) {
		t.Fatalf("expect %v got %v", ts, extractedTs)
	}
}

func BenchmarkNewID(b *testing.B) {
	process := NewProcess(1)
	t := time.Now()

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		id := process.NewID(2, t)
		_ = id
		t = t.Add(time.Nanosecond)
	}
}
