package testcore

import (
	"path/filepath"
	"runtime"
)

func ProjectFile(p string) string {
	_, filename, _, _ := runtime.Caller(0)
	wd := filepath.Dir(filename)
	out := filepath.Join(wd, "./../../..", p)
	return out
}
