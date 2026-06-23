package ioutils

import (
	"io"
	"os"
	"path/filepath"

	"go.podman.io/storage/pkg/ioutils/atomicwriter"
)

type AtomicFileWriterOptions = atomicwriter.AtomicFileWriterOptions

type CommittableWriter = atomicwriter.CommittableWriter

func SetDefaultOptions(opts AtomicFileWriterOptions) {
	atomicwriter.SetDefaultOptions(opts)
}

func NewAtomicFileWriterWithOpts(filename string, perm os.FileMode, opts *AtomicFileWriterOptions) (CommittableWriter, error) {
	return atomicwriter.NewAtomicFileWriterWithOpts(filename, perm, opts)
}

func NewAtomicFileWriter(filename string, perm os.FileMode) (CommittableWriter, error) {
	return atomicwriter.NewAtomicFileWriter(filename, perm)
}

func AtomicWriteFileWithOpts(filename string, data []byte, perm os.FileMode, opts *AtomicFileWriterOptions) error {
	return atomicwriter.AtomicWriteFileWithOpts(filename, data, perm, opts)
}

func AtomicWriteFile(filename string, data []byte, perm os.FileMode) error {
	return atomicwriter.AtomicWriteFile(filename, data, perm)
}

// AtomicWriteSet is used to atomically write a set
// of files and ensure they are visible at the same time.
// Must be committed to a new directory.
type AtomicWriteSet struct {
	root string
}

// NewAtomicWriteSet creates a new atomic write set to
// atomically create a set of files. The given directory
// is used as the base directory for storing files before
// commit. If no temporary directory is given the system
// default is used.
func NewAtomicWriteSet(tmpDir string) (*AtomicWriteSet, error) {
	td, err := os.MkdirTemp(tmpDir, "write-set-")
	if err != nil {
		return nil, err
	}

	return &AtomicWriteSet{
		root: td,
	}, nil
}

// WriteFile writes a file to the set, guaranteeing the file
// has been synced.
func (ws *AtomicWriteSet) WriteFile(filename string, data []byte, perm os.FileMode) error {
	f, err := ws.FileWriter(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

type syncFileCloser struct {
	*os.File
}

func (w syncFileCloser) Close() error {
	if !atomicwriter.DefaultOptions().NoSync {
		return w.File.Close()
	}
	err := dataOrFullSync(w.File)
	if err1 := w.File.Close(); err == nil {
		err = err1
	}
	return err
}

// FileWriter opens a file writer inside the set. The file
// should be synced and closed before calling commit.
func (ws *AtomicWriteSet) FileWriter(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
	f, err := os.OpenFile(filepath.Join(ws.root, name), flag, perm)
	if err != nil {
		return nil, err
	}
	return syncFileCloser{f}, nil
}

// Cancel cancels the set and removes all temporary data
// created in the set.
func (ws *AtomicWriteSet) Cancel() error {
	return os.RemoveAll(ws.root)
}

// Commit moves all created files to the target directory. The
// target directory must not exist and the parent of the target
// directory must exist.
func (ws *AtomicWriteSet) Commit(target string) error {
	return os.Rename(ws.root, target)
}

// String returns the location the set is writing to.
func (ws *AtomicWriteSet) String() string {
	return ws.root
}
