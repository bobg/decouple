package m

import (
	"context"
	"io"
	"os"
)

// In these tests,
// a parameter named r can be an io.Reader,
// and a parameter named rc can be an io.ReadCloser.
// Other parameter names cannot be decoupled.

func F1(r *os.File, n int) ([]byte, error) {
	if true { // This exercises the *ast.BlockStmt typeswitch clause.
		buf := make([]byte, n)
		n, err := r.Read(buf)
		return buf[:n], err
	}
	return nil, nil
}

func F2(r *os.File) ([]byte, error) {
	return io.ReadAll(r)
}

func F3(lf *io.LimitedReader) ([]byte, int64, error) {
	b, err := io.ReadAll((lf)) // extra parens sic
	return b, lf.N, err
}

func F4(f *os.File) ([]byte, error) {
	var f2 *os.File = f // Some day perhaps decouple will be clever enough to know that f and f2 can both be io.Readers.
	return io.ReadAll(f2)
}

func F5(r *os.File) ([]byte, error) {
	var f2 io.Reader = r
	return io.ReadAll(f2)
}

func F6(f *os.File) ([]byte, error) {
	return F7(f)
}

func F7(rc *os.File) ([]byte, error) {
	defer rc.Close()
	goto LABEL
LABEL:
	return io.ReadAll(rc)
}

type intErface int

func (i intErface) Read([]byte) (int, error) {
	return 0, nil
}

func F8(r intErface) ([]byte, error) {
	return io.ReadAll(r)
}

func F9(i intErface) int {
	return int(i) + 1
}

func F10(r *os.File) ([]byte, error) {
	var r2 io.Reader
	r2 = r // separate non-defining assignment line sic
	return io.ReadAll(r2)
}

func F11(r *os.File) ([]byte, error) {
	switch r {
	case r:
		return io.ReadAll(r)
	default:
		return nil, nil
	}
}

func F12(f *os.File) ([]byte, error) {
	var f2 os.File
	switch f2 {
	case *f:
		return io.ReadAll(f)
	default:
		return nil, nil
	}
}

func F13(ctx context.Context, ch chan<- io.Reader, r *os.File) {
	for {
		select {
		case <-ctx.Done():
			return
		case ch <- r:
			// do nothing
		}
	}
}

func F14(r *os.File) []io.Reader {
	return []io.Reader{r}
}

type boolErface bool

func (b boolErface) Read([]byte) (int, error) {
	return 0, nil
}

func F15(b boolErface) ([]byte, error) {
	switch {
	case bool(b):
		return io.ReadAll(b)
	default:
		return nil, nil
	}
}

func F16(b boolErface) ([]byte, error) {
	switch {
	case true:
		if bool(b) {
			return io.ReadAll(b)
		}
	}
	return nil, nil
}

func F17(r *os.File) ([]byte, error) {
	var x io.Reader
	if r == x {
		return nil, nil
	}
	return io.ReadAll(r)
}

func F18(f *os.File) ([]byte, error) {
	if f == nil {
		return nil, nil
	}
	return io.ReadAll(f)
}

type funcErface func()

func (f funcErface) Read([]byte) (int, error) {
	return 0, nil
}

func F19(f funcErface) ([]byte, error) {
	f()
	return io.ReadAll(f)
}

func F20(r *os.File) func([]byte) (int, error) {
	return r.Read
}

func F21(f *os.File) map[*os.File]int {
	return map[*os.File]int{f: 0}
}

func F22(rc *os.File) map[io.ReadCloser]int {
	return map[io.ReadCloser]int{rc: 0}
}

func F23(f *os.File) *os.File {
	return f
}

func F24(rc *os.File) io.ReadCloser {
	return rc
}

func F25(r *os.File) ([]byte, error) {
	return func() ([]byte, error) {
		return io.ReadAll(r)
	}()
}

func F26(f *os.File) io.Reader {
	return func() *os.File {
		return f
	}()
}

func F27(r *os.File) (data []byte, err error) {
	ch := make(chan struct{})
	go func() {
		data, err = io.ReadAll(r)
		close(ch)
	}()
	<-ch
	return
}

func F28(r *os.File) map[int]io.Reader {
	return map[int]io.Reader{7: r}
}

func F29(r io.ReadCloser) ([]byte, error) {
	return io.ReadAll(r)
}

func F30(x io.ReadCloser) ([]byte, error) {
	defer x.Close()
	return io.ReadAll(x)
}

func F31(r *os.File) io.Reader {
	x := []io.Reader{r}
	return x[0]
}
