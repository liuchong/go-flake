package flake

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const (
	workerIDBits       = uint64(10)
	maxWorkerID        = int64(-1) ^ (int64(-1) << workerIDBits)
	sequenceBits       = uint64(13) // do not use standard 12 bits
	workerIDShift      = sequenceBits
	timestampLeftShift = sequenceBits + workerIDBits
	sequenceMask       = int64(-1) ^ (int64(-1) << sequenceBits)
)

// FlakeID is short for (a simple) flake ID.
type FlakeID uint64

// id format:
// timestampBits(41) | workerBits(10) | sequenceBits(13)

// Generator generates new FlakeID
type Gen struct {
	sync.Mutex
	seq      int64
	ts       int64 // the last timestamp in milliseconds
	fepoch   int64
	workerID int64 // worker id  0 <= workerID <= maxWorkerID
}

func NewGen(workerID, fepoch int64) (*Gen, error) {
	if workerID < 0 || workerID > maxWorkerID {
		return nil, fmt.Errorf("worker id must be between 0 and %d, actual got %d",
			maxWorkerID, workerID)
	}

	now, _ := getTsInfo()
	if now < fepoch {
		return nil, fmt.Errorf("fepoch %d is moving backwards", fepoch)
	}

	if fepoch <= 0 {
		// set default epoch 1234567891011
		// 2009-02-13T23:31:31.011Z
		fepoch = int64(1234567891011)
	}

	return &Gen{
		seq:      -1,
		ts:       -1,
		fepoch:   fepoch,
		workerID: workerID,
	}, nil
}

// NextID returns the next unique id.
func (g *Gen) NextID() FlakeID {
	g.Lock()
	defer g.Unlock()

	ts, rem := getTsInfo()
	lastTs := g.ts
	seq := g.seq

	switch {
	// ts is never less than lastTs
	case ts == lastTs:
		seq = (seq + 1) & sequenceMask
		if seq == 0 {
			for ts <= lastTs {
				time.Sleep(time.Duration(rem))
				ts, rem = getTsInfo()
			}
		}
	default:
		seq = 0
	}

	g.ts = ts
	g.seq = seq

	return FlakeID(
		(0 |
			// timestamp
			(ts-g.fepoch)<<timestampLeftShift) |
			// workid
			(g.workerID << workerIDShift) |
			// sequence
			seq,
	)
}

// GenMulti returns next n ids where n is given by parameter.
func (g *Gen) GenMulti(n uint) []byte {
	b := make([]byte, n*8)
	for i := uint(0); i < n; i++ {
		id := g.NextID()
		off := i * 8
		b[off+0] = byte(id >> 56)
		b[off+1] = byte(id >> 48)
		b[off+2] = byte(id >> 40)
		b[off+3] = byte(id >> 32)
		b[off+4] = byte(id >> 24)
		b[off+5] = byte(id >> 16)
		b[off+6] = byte(id >> 8)
		b[off+7] = byte(id)
	}
	return b
}

// ToBytes convert id to byte array.
func (id *FlakeID) ToBytes() []byte {
	b := make([]byte, 8)

	b[0] = byte(*id >> 56)
	b[1] = byte(*id >> 48)
	b[2] = byte(*id >> 40)
	b[3] = byte(*id >> 32)
	b[4] = byte(*id >> 24)
	b[5] = byte(*id >> 16)
	b[6] = byte(*id >> 8)
	b[7] = byte(*id)

	return b
}

// ToString encode FlakeID to URL-compatible base64 string.
func (id FlakeID) ToString() string {
	bs := id.ToBytes()
	return base64.URLEncoding.EncodeToString(bs)
}

// FromString decode URL-compatible base64 string to FlakeID.
func (id *FlakeID) FromString(s string) error {
	bs, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return err
	}

	*id = FlakeID(
		(int64(bs[0]) << 56) |
			(int64(bs[1]) << 48) |
			(int64(bs[2]) << 40) |
			(int64(bs[3]) << 32) |
			(int64(bs[4]) << 24) |
			(int64(bs[5]) << 16) |
			(int64(bs[6]) << 8) |
			int64(bs[7]),
	)

	return nil
}

// MarshalJSON automatically convert id to string for JSON.
func (id FlakeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.ToString())
}

// UnmarshalJSON convert JSON string to FlakeID.
func (id *FlakeID) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}

	return id.FromString(s)
}

func getTsInfo() (milliseconds, remain int64) {
	nano := time.Now().UnixNano()

	return nano / 1e6, 1e6 - nano%1e6
}
