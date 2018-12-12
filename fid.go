package fid

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// FID is short for (a simple) flake ID.
type FID uint64

// id format:
// timestampBits(41) | workerBits(10) | sequenceBits(13)

const (
	workerIDBits = uint64(10)

	maxWorkerID = int64(-1) ^ (int64(-1) << workerIDBits)

	sequenceBits       = uint64(13)
	workerIDShift      = sequenceBits
	timestampLeftShift = sequenceBits + workerIDBits
	sequenceMask       = int64(-1) ^ (int64(-1) << sequenceBits)

	// 2009-02-13T23:31:31.011Z
	twepoch = int64(1234567891011)
)

// Flags
var (
	workerID int64      // worker id  0 <= workerID <= maxWorkerID
	lastTs   int64 = -1 // the last timestamp in milliseconds
)

var (
	mu  sync.Mutex
	seq int64
)

// Config (ure) this fid generator.
func Config(id int64) {
	workerID = id
	if workerID < 0 || workerID > maxWorkerID {
		log.Fatalf("worker id must be between 0 and %d", maxWorkerID)
	}
}

func getTsInfo() (milliseconds, remain int64) {
	nano := time.Now().UnixNano()

	return nano / 1e6, 1e6 - nano%1e6
}

// NextID returns the next unique id.
func NextID() (FID, error) {
	mu.Lock()
	defer mu.Unlock()

	// ts := milliseconds()
	ts, rem := getTsInfo()

	switch {
	case ts < lastTs:
		return 0, fmt.Errorf("time is moving backwards, waiting until %d", lastTs)
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

	lastTs = ts

	return FID(

		(0 |
			// timestamp
			(ts-twepoch)<<timestampLeftShift) |
			// workid
			(workerID << workerIDShift) |
			// sequence
			seq,
	), nil
}

// GenMulti returns next n ids where n is given by parameter.
func GenMulti(n uint) ([]byte, error) {
	b := make([]byte, n*8)
	for i := uint(0); i < n; i++ {
		id, err := NextID()
		if err != nil {
			return nil, err
		}

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
	return b, nil
}

// ToBytes convert id to byte array.
func (id *FID) ToBytes() []byte {
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

// ToString encode FID to URL-compatible base64 string.
func (id FID) ToString() string {
	bs := id.ToBytes()
	return base64.URLEncoding.EncodeToString(bs)
}

// FromString decode URL-compatible base64 string to FID.
func (id *FID) FromString(s string) error {
	bs, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return err
	}

	*id = FID(
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
func (id FID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.ToString())
}

// UnmarshalJSON convert JSON string to FID.
func (id *FID) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)

	if err != nil {
		return err
	}

	return id.FromString(s)
}
