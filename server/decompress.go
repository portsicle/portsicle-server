package server

import (
	"bytes"
	"io"
	"log"

	"github.com/pierrec/lz4"
)

// lz4decompress returns the Lempel-Ziv-4 uncompressed in bytes.
func lz4decompress(in []byte) []byte {
	r := bytes.NewReader(in)
	w := &bytes.Buffer{}
	zr := lz4.NewReader(r) // an LZ4 reader that decompresses

	_, err := io.Copy(w, zr) // Copies data through the decompressor
	if err != nil {
		log.Print("Error decompressing response body")
		return nil
	}

	return w.Bytes()
}
