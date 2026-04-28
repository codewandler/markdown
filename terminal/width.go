package terminal

import (
	"io"
	"os"
)

func detectWrapWidth(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok {
		return 0
	}
	return terminalWidth(f)
}
