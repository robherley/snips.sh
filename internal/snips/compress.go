package snips

import "encoding/binary"

// IsZSTDCompressed checks if the data starts with the zstd magic number.
// https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md#zstandard-frames
func IsZSTDCompressed(data []byte) bool {
	return len(data) > 4 && binary.BigEndian.Uint32(data) == 0x28B52FFD
}
