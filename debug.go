package decouple

import (
	"fmt"
	"os"
	"strings"
)

func (a *analyzer) debugf(format string, args ...any) {
	if !a.debug {
		return
	}
	s := fmt.Sprintf(format, args...)
	s = strings.TrimRight(s, "\r\n")
	if a.level > 0 {
		fmt.Fprint(os.Stderr, strings.Repeat("  ", a.level))
	}
	fmt.Fprintln(os.Stderr, s)
}
