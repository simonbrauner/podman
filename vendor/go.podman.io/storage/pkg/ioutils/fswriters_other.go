//go:build !linux

package ioutils

import (
	"os"
)

func dataOrFullSync(f *os.File) error {
	return f.Sync()
}
