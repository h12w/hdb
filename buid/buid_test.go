package buid

import (
	"fmt"
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

	for i := 0; i <= 65535; i++ {
		id = process.NewID(2, ts)
		_, key := id.Split()
		if int(key.Counter()) != i {
			t.Fatalf("expect counter %d got %d", i, key.Counter())
		}
		if !ts.Equal(id.Time()) {
			t.Fatalf("expect time %v got %v", ts, id.Time())
		}
	}

	// get the first ID based on the overflowed counter
	id = process.NewID(2, ts)
	_, key := id.Split()
	if key.Counter() != 0 {
		t.Fatalf("expect 0 got %d", key.Counter())
	}

	expectedTs := externalTime(process.t)
	if !expectedTs.After(ts) {
		t.Fatal("expect the ts proceed")
	}
	if extractedTs := id.Time(); !extractedTs.Equal(expectedTs) {
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

func TestKeyCounterInc(t *testing.T) {
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

func TestUniqueness(t *testing.T) {
	m := make(map[ID]bool)
	process := NewProcess(12)
	for i := 0; i < 1000000; i++ {
		id := process.NewID(1, time.Now())
		if m[id] {
			t.Fatal(i, "duplicates detected")
		}
		m[id] = true
	}
}

func BenchmarkMaxCounter(b *testing.B) {
	process := NewProcess(12)
	b.RunParallel(func(pb *testing.PB) {
		c := uint16(0)
		for pb.Next() {
			id := process.NewID(1, time.Now())
			_, key := id.Split()
			if key.Counter() > c {
				c = key.Counter()
			}
		}
		fmt.Println(c)
	})
}
