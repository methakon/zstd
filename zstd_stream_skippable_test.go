package zstd

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"testing"
)

// skippableFrame returns a valid zstd skippable frame carrying 4 bytes of user data.
// https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md#skippable-frames
func skippableFrame(userValue uint32) []byte {
	skipFrame := make([]byte, 12)
	binary.LittleEndian.PutUint32(skipFrame, 0x184D2A50) // user-defined skippable magic
	binary.LittleEndian.PutUint32(skipFrame[4:], 4)      // frame content size
	binary.LittleEndian.PutUint32(skipFrame[8:], userValue)
	return skipFrame
}

// TestDecompressTrailingSkippableFrame verifies that one-shot Decompress accepts a
// valid zstd stream followed by a trailing skippable frame. The payload is highly
// compressible, so the compressed input is tiny relative to its decompressed size;
// that is the shape where the streaming fallback previously stopped after the first
// frame and reported unexpected EOF on the trailing skippable frame.
// Regression context: https://github.com/DataDog/zstd/issues/129
func TestDecompressTrailingSkippableFrame(t *testing.T) {
	totalSize := 1_000_001
	rawData := make([]byte, totalSize) // all zeros: highly compressible

	compressed, err := Compress(nil, rawData)
	if err != nil {
		t.Fatal(err)
	}
	compressed = append(compressed, skippableFrame(12345)...)

	decompressed, err := Decompress(nil, compressed)
	if err != nil {
		t.Fatalf("Decompress: %s", err)
	}
	if !bytes.Equal(rawData, decompressed) {
		t.Fatal("decompressed data does not match original")
	}
}

// TestReaderTrailingSkippableFrame verifies the same through the streaming reader.
func TestReaderTrailingSkippableFrame(t *testing.T) {
	totalSize := 1_000_001
	rawData := make([]byte, totalSize)
	r := NewRandBytes()
	if _, err := io.ReadFull(r, rawData); err != nil && err != io.EOF {
		t.Fatal(err)
	}

	compressed, err := Compress(nil, rawData)
	if err != nil {
		t.Fatal(err)
	}
	compressed = append(compressed, skippableFrame(12345)...)

	reader := NewReader(bytes.NewReader(compressed))
	defer reader.Close()
	decompressed, err := ioutil.ReadAll(reader)
	if err != nil {
		t.Fatalf("stream reader: %s", err)
	}
	if !bytes.Equal(rawData, decompressed) {
		t.Fatal("decompressed data does not match original")
	}
}
