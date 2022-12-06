package m

import (
	"io"
	"os"
)

func Yes1(f *os.File, n int) ([]byte, error) {
	if true { // This exercises the *ast.BlockStmt typeswitch clause.
		buf := make([]byte, n)
		n, err := f.Read(buf)
		return buf[:n], err
	}
	return nil, nil
}

func Yes2(f *os.File) ([]byte, error) {
	return io.ReadAll(f)
}

func No3(lf *io.LimitedReader) ([]byte, int64, error) {
	b, err := io.ReadAll(lf)
	return b, lf.N, err
}

func No4(f *os.File) ([]byte, error) {
	var f2 *os.File = f // Some day perhaps decouple will be clever enough to know that f and f2 can both be io.Readers.
	return io.ReadAll(f2)
}

func Yes5(f *os.File) ([]byte, error) {
	var f2 io.Reader = f
	return io.ReadAll(f2)
}

func No6(f *os.File) ([]byte, error) {
	return Yes7(f)
}

func Yes7(f *os.File) ([]byte, error) {
	defer f.Close()
	goto LABEL
LABEL:
	return io.ReadAll(f)
}

type intErface int

func (i intErface) x() {}

func Yes8(i intErface) {
	i.x()
}

func No9(i intErface) int {
	var j int
	j += int(i)
	return j
}

func No10(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

func Yes11(f *os.File) ([]byte, error) {
	return io.ReadAll((f))
}

func No12(f *os.File) ([]byte, error) {
	f2 := *f
	return io.ReadAll(&f2)
}

func Yes13(f *os.File) ([]byte, error) {
	var r io.Reader
	r = f
	return io.ReadAll(r)
}

func Yes14(f *os.File) ([]byte, error) {
	switch f {
	case f:
		return io.ReadAll(f)
	default:
		return nil, nil
	}
}

func No15(f *os.File) ([]byte, error) {
	var f2 os.File
	switch f2 {
	case *f:
		return io.ReadAll(f)
	default:
		return nil, nil
	}
}
