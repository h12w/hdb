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

Definitions:

   | item         | type   | description                                                        |
   |--------------|--------|--------------------------------------------------------------------|
   | shard-index  | uint16 | the index of the shard for storing the data associated to the BUID |
   | hours        | uint32 | hours from bespoke epoch (490,293 years, should be enough :-)      |
   | minutes      | uint6  | 0-59 minutes within an hour                                        |
   | seconds      | uint6  | 0-59 seconds within a minute                                       |
   | microseconds | uint16 | 0-999999 microseconds within a second                              |
   | process      | uint16 | a unique process on a specific node                                |
   | counter      | uint16 | cyclic counter for within each microsecond                         |

*/
package buid

import (
	"math"
	"sync"
	"time"
)

type (
	// ID is BUID
	ID [16]byte
	// Shard part of the BUID
	Shard [8]byte
	// Key part of the BUID
	Key [8]byte

	// Process represents a unique process on a specific node
	Process struct {
		id      uint16
		counter uint32
		t       int64
		mu      sync.Mutex
	}
)

const (
	secondInMicroseconds = 1000000
	minuteInMicroseconds = 60 * 1000000
	hourInMicroseconds   = 60 * 60 * 1000000
)

// Epoch is the bespoke epoch of BUID in Unix Epoch in microseconds
var Epoch = time.Date(2017, 10, 24, 0, 0, 0, 0, time.UTC).UnixNano() / 1000

// internalTime returns internal epoch time in microseconds
func internalTime(t time.Time) int64 {
	return t.UnixNano()/1000 - Epoch
}
func externalTime(t int64) time.Time {
	return time.Unix(0, (Epoch+t)*1000).UTC()
}

// NewProcess returns a new Process object for id
func NewProcess(id uint16) *Process {
	// the generator needs to wait a microsecond to avoid
	// possible conflict caused by restarting within a microsecond
	t := time.Now()
	for {
		now := time.Now()
		if now.Sub(t) > time.Microsecond {
			t = now
			break
		}
	}

	return &Process{
		id: id,
		t:  internalTime(t),
	}
}

// NewID generates a new BUID from a shard index and a timestamp
func (p *Process) NewID(shard uint16, timestamp time.Time) ID {
	ts := internalTime(timestamp)
	counter := uint16(0)
	p.mu.Lock()
	if ts > p.t {
		p.t = ts
		p.counter = 0
	} else { // if ts == n.t || ts < n.t (same time or the clock is rewinded)
		if p.counter > math.MaxUint16 { // is full
			for {
				now := internalTime(time.Now())
				if now > p.t {
					p.t = now
					p.counter = 0
					break
				}
			}
		}
		counter = uint16(p.counter)
		p.counter++
	}
	t := p.t
	p.mu.Unlock()

	var (
		hour    = uint32(t / hourInMicroseconds)
		minute  = uint8((t % hourInMicroseconds) / minuteInMicroseconds)
		second  = uint8((t % minuteInMicroseconds) / secondInMicroseconds)
		micro   = uint32(t % secondInMicroseconds)
		process = p.id
	)

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

// Time returns the embedded timestamp
func (id ID) Time() time.Time {
	var (
		hour = (uint32(id[4]) << 24) |
			(uint32(id[5]) << 16) |
			(uint32(id[6]) << 8) |
			uint32(id[7])
		minute = (id[8] & 0xfc) >> 2
		second = ((id[8] & 0x03) << 4) | (id[9] >> 4)
		micro  = (uint32(id[9]&0x0f) << 16) |
			(uint32(id[10]) << 8) |
			uint32(id[11])
		t = int64(hour)*hourInMicroseconds +
			int64(minute)*minuteInMicroseconds +
			int64(second)*secondInMicroseconds +
			int64(micro)
	)
	return externalTime(t)
}

// Split splits BUID to Shard and Key
func (id ID) Split() (Shard, Key) {
	var shard Shard
	var key Key
	copy(shard[:], id[:8])
	copy(key[:], id[8:])
	return shard, key
}

// Index returns the embedded shard index
func (s Shard) Index() uint16 {
	return (uint16(s[0]) << 8) | uint16(s[1])
}

// Time returns the embedded hours in time.Time
func (s Shard) Time() time.Time {
	hour := (uint32(s[4]) << 24) |
		(uint32(s[5]) << 16) |
		(uint32(s[6]) << 8) |
		uint32(s[7])
	t := int64(hour) * hourInMicroseconds
	return externalTime(t)
}

// Time returns the embedded time in time.Duration
func (k Key) Time() time.Duration {
	minute := (k[0] & 0xfc) >> 2
	second := ((k[0] & 0x03) << 4) | (k[1] >> 4)
	micro := (uint32(k[1]&0x0f) << 16) |
		(uint32(k[2]) << 8) |
		uint32(k[3])
	return (time.Duration(minute)*minuteInMicroseconds +
		time.Duration(second)*secondInMicroseconds +
		time.Duration(micro)) * 1000
}

// Process returns the embedded process ID
func (k Key) Process() uint16 {
	return (uint16(k[4]) << 8) | uint16(k[5])
}

// Counter returns the embedded counter part of the key
func (k Key) Counter() uint16 {
	return (uint16(k[6]) << 8) | uint16(k[7])
}
