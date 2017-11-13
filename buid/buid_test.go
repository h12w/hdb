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

func TestCounterReset(t *testing.T) {
	process := NewProcess(1)
	ts := time.Now().UTC().Truncate(time.Microsecond)
	var id ID
	for i := 0; i < 5; i++ {
		id = process.NewID(2, ts)
	}
	_, key := id.Split()
	if key.Counter() == 0 {
		t.Fatal("expect counter > 0")
	}

	ts = ts.Add(-time.Millisecond)
	id = process.NewID(2, ts)
	_, key = id.Split()
	if key.Counter() == 0 {
		t.Fatal("expect counter still > 0, not reset")
	}

	ts = ts.Add(time.Millisecond)
	id = process.NewID(2, ts)
	_, key = id.Split()
	if key.Counter() == 0 {
		t.Fatal("expect counter still > 0, not reset")
	}

	ts = ts.Add(time.Millisecond)
	id = process.NewID(2, ts)
	_, key = id.Split()
	if key.Counter() > 0 {
		t.Fatal("expect counter is reset")
	}
}

func TestCounterOverflow(t *testing.T) {
	var id ID
	process := NewProcess(1)
	ts := time.Now().UTC().Truncate(time.Microsecond)

	// reset process ts
	{
		processTs := externalTime(process.t)
		for processTs.Equal(ts) {
			id = process.NewID(2, ts)
			processTs = externalTime(process.t)
		}
	}

	for i := 0; i < 65536; i++ {
		id = process.NewID(2, ts)
	}
	expectedTs := externalTime(process.t)
	if !expectedTs.After(ts) {
		t.Fatal("expect the ts proceed")
	}
	if extractedTs := id.Time(); !extractedTs.Equal(expectedTs) {
		t.Fatalf("expect %v got %v", ts, extractedTs)
	}
	_, key := id.Split()
	if key.Counter() != 0 {
		t.Fatalf("expect 0 got %d", key.Counter())
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

func TestShardIndex(t *testing.T) {
	process := NewProcess(1)
	id := process.NewID(42, time.Now())
	shard, _ := id.Split()
	if shard.Index() != 42 {
		t.Fatalf("expect 42 got %d", shard.Index())
	}
}

func TestShardTime(t *testing.T) {
	ts := time.Now().UTC()
	hour := ts.Truncate(time.Hour)
	process := NewProcess(1)
	id := process.NewID(42, ts)
	shard, _ := id.Split()
	if !shard.Time().Equal(hour) {
		t.Fatalf("expect %v got %v", hour, shard.Time())
	}
}

func TestKeyTime(t *testing.T) {
	process := NewProcess(1)
	ts := externalTime(process.t)
	id := process.NewID(42, ts)
	_, key := id.Split()
	expected := ts.Sub(ts.Truncate(time.Hour))
	if key.Time() != expected {
		t.Fatalf("expect %v got %v", expected, key.Time())
	}
}

func TestKeyProcess(t *testing.T) {
	ts := time.Now().UTC().Truncate(time.Microsecond)
	process := NewProcess(12)
	id := process.NewID(42, ts)
	_, key := id.Split()
	if key.Process() != 12 {
		t.Fatalf("expect 12 got %v", key.Process())
	}
}

func TestKeyCounter(t *testing.T) {
	ts := time.Now().UTC().Truncate(time.Microsecond)
	process := NewProcess(12)
	var id ID
	for i := 0; i < 23; i++ {
		id = process.NewID(42, ts)
	}
	_, key := id.Split()
	if key.Counter() != 22 {
		t.Fatalf("expect 22 got %v", key.Counter())
	}
}
