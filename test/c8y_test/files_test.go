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
		panic(errors.Wrap(err, "Error creating dummy file"))
	}

	defer f.Close()

	f.WriteString(contents)

	if err := f.Sync(); err != nil {
		panic(errors.Wrap(err, "Failed to fill file with dummy information"))
	}

	filepath = f.Name()
	return
}

func NewDummyFileWithSize(name string, size int64) (filepath string) {
	if name == "" {
		name = "test-dummy-dummy"
	}

	if size < 0 {
		size = 10_000_000
	}

	f, err := os.Create(name)
	if err != nil {
		panic(errors.Wrap(err, "Error creating dummy file"))
	}

	defer f.Close()

	if err := f.Truncate(size); err != nil {
		panic(errors.Wrap(err, "Failed to fill file with dummy information"))
	}

	if err := f.Sync(); err != nil {
		panic(errors.Wrap(err, "Failed to fill file with dummy information"))
	}

	filepath = f.Name()
	return
}
