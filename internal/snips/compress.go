package snips

import (
	"encoding/binary"

	"github.com/klauspost/compress/zstd"
)

// IsZSTDCompressed checks if the data starts with the zstd magic number.
// https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md#zstandard-frames
func IsZSTDCompressed(data []byte) bool {
	return len(data) > 4 && binary.BigEndian.Uint32(data) == 0x28B52FFD
}

func EncodeContent(content []byte, compress bool) ([]byte, error) {
	if !compress {
		return content, nil
	}

	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	defer encoder.Close()

	return encoder.EncodeAll(content, nil), nil
}

func DecodeContent(content []byte) ([]byte, error) {
	if !IsZSTDCompressed(content) {
		return content, nil
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()

	return decoder.DecodeAll(content, nil)
}
