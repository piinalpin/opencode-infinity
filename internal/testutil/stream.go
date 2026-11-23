package testutil

import (
	"context"
	"errors"
	"io"
	"net/http"
)

const MaxFrameBytes = 64 * 1024

// WriteFragments memotong Write, bukan menjanjikan batas paket atau Read TCP.
func WriteFragments(ctx context.Context, w http.ResponseWriter, data []byte, sizes []int) error {
	if len(data) > MaxFrameBytes {
		return errors.New("frame fixture melewati batas")
	}
	if len(sizes) == 0 {
		sizes = []int{MaxFrameBytes}
	}
	for _, n := range sizes {
		if n <= 0 {
			return errors.New("ukuran fragmen harus positif")
		}
	}
	for offset, index := 0, 0; offset < len(data); index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		n := min(sizes[index%len(sizes)], len(data)-offset)
		written, err := w.Write(data[offset : offset+n])
		if err != nil {
			return err
		}
		if written != n {
			return io.ErrShortWrite
		}
		if err := http.NewResponseController(w).Flush(); err != nil {
			return err
		}
		offset += n
	}
	return nil
}
