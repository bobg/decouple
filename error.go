package decouple

import "fmt"

type derr struct {
	error
}

func errf(format string, args ...any) error {
	return derr{error: fmt.Errorf(format, args...)}
}
