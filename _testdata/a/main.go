package a

import "io"

func X(r io.ReadCloser) ([]byte, error) {
	return io.ReadAll(r)
}
