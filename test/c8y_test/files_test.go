package c8y_test

import (
	"os"

	"github.com/pkg/errors"
)

func NewDummyFile(name string, contents string) (filepath string) {
	if name == "" {
		name = "test-dummy-dummy"
	}
	f, err := os.Create(name)
	if err != nil {
		panic(errors.Wrap(err, "Error creating dummyfile"))
	}

	defer f.Close()

	f.WriteString(contents)

	if err := f.Sync(); err != nil {
		panic(errors.Wrap(err, "Failed to fill file with dummy information"))
	}

	filepath = f.Name()
	return
}
