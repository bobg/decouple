package m

import (
	"io"
	"os"
)

func FirstN(f *os.File, n int) ([]byte, error) {
	buf := make([]byte, n)
	n, err := f.Read(buf)
	return buf[:n], err
}

func All(ff *os.File) ([]byte, error) {
	return io.ReadAll(ff)
}
