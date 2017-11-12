/*
Package buid provides Bipartite Unique Identifier (BUID)

A BUID is a 128-bit unique ID composed of two 64-bit parts: shard and key.

It is not only a unique ID, but also contains the sharding information, so that
the messages with the same BUID could be stored together within the same DB shard.

Also, when a message is stored in a shard, the shard part of the BUID can be
trimmed off to save the space, and only the key part needs to be stored as the
primary key.

Bigendian is chosen to make each part byte-wise lexicographic sortable.

BUID = shard key .

shard:

    0             1               2               3
    7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |           shard-index         |            reserved           |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                  hours (from bespoke epoch)                   |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

key:

    0             1               2               3
    7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |  minutes  |  seconds  |            microseconds               |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |            process            |            counter            |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

- shard-index (uint16): the index of the shard for storing the data associated to the BUID
- hours (uint32): hours from bespoke epoch time
- minutes (uint6): 0-59 minutes within an hour
- seconds (uint6): 0-59 seconds within a minute
- microseconds (uint16): 0-999999 microseconds within a second
- process (uint16): a unique process on a specific node
- counter (uint16): cyclic counter for within each microsecond

*/
package buid

import (
	"sync"
	"time"
)

type (
	// ID is BUID
	ID    [16]byte
	Shard [8]byte
	Key   [8]byte

	// Process represents a unique process on a specific node
	Process struct {
		id      uint16
		counter uint16
		t       time.Time
		mu      sync.Mutex
	}
)

// Epoch is the bespoke epoch of BUID
var Epoch = time.Date(2017, 10, 24, 0, 0, 0, 0, time.UTC)

// NewProcess returns a new Process object for id
func NewProcess(id uint16) *Process {
	return &Process{
		id: id,
		t:  time.Now(),
	}
}

// the generator needs a microsecond to be initialized to avoid conflict caused by restarting within an microsecond

// NewID generates a new BUID from a shard index and a timestamp
func (n *Process) NewID(shard uint16, ts time.Time) ID {
	counter := uint16(0)
	n.mu.Lock()
	if ts.After(n.t) {
		n.t = ts
		n.counter = 0
	} else { // if ts == n.t || ts < n.t (same time or the clock is rewinded)
		if n.counter == 0 { // is full
			for {
				now := time.Now()
				if now.After(n.t) {
					n.t = now
					break
				}
			}
		}
		counter = n.counter
		n.counter++
	}
	t := n.t
	n.mu.Unlock()

	// TODO: can be optimize with customized code
	minute := uint8(t.Minute())
	second := uint8(t.Second())
	micro := uint32(t.Nanosecond() / 1000)
	hour := uint32(t.Sub(Epoch).Hours())
	process := n.id
	// END TODO

	return ID{
		byte(shard >> 8), byte(shard),
		0, 0, // reserved
		byte(hour >> 24), byte(hour >> 16), byte(hour >> 8), byte(hour),
		((minute & 0x3f) << 2) | ((second & 0x30) >> 4),
		((second & 0x0f) << 4) | byte(micro>>16),
		byte(micro >> 8), byte(micro),
		byte(process >> 8), byte(process),
		byte(counter >> 8), byte(counter),
	}
}

// Split splits BUID to Shard and Key
func (id *ID) Split() (Shard, Key) {
	var shard Shard
	var key Key
	copy(shard[:], (*id)[:8])
	copy(key[:], (*id)[8:])
	return shard, key
}
