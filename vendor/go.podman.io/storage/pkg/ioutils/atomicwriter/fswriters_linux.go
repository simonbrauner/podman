package atomicwriter

import (
	"golang.org/x/sys/unix"
)

func (w *atomicFileWriter) postDataWrittenSync() error {
	if w.noSync {
		return nil
	}
	return unix.Fdatasync(int(w.f.Fd()))
}

func (w *atomicFileWriter) preRenameSync() error {
	// On Linux data can be reliably flushed to media without metadata, so defer
	return nil
}
