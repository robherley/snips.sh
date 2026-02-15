package snips

import (
	"encoding/binary"
	"time"

	"github.com/klauspost/compress/zstd"
)

type Revision struct {
	ID        int64
	FileID    string
	CreatedAt time.Time
	RawDiff   []byte // may be zstd-compressed
	Size      uint64 // file size after this revision
	Type      string // file type after this revision
}

func (r *Revision) GetDiff() ([]byte, error) {
	if !r.isCompressed() {
		return r.RawDiff, nil
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	defer decoder.Close()
	decodedBytes, err := decoder.DecodeAll(r.RawDiff, nil)

	return decodedBytes, err
}

func (r *Revision) SetDiff(in []byte, compress bool) error {
	if !compress {
		r.RawDiff = in
		return nil
	}

	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return err
	}

	r.RawDiff = encoder.EncodeAll(in, nil)
	return encoder.Close()
}

func (r *Revision) isCompressed() bool {
	return len(r.RawDiff) > 4 && binary.BigEndian.Uint32(r.RawDiff) == 0x28B52FFD
}
