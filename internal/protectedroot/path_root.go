package protectedroot

import (
	"encoding/json"
	"errors"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
)

type PathRoot struct {
	path string
}

func NewPathRoot(path string) (PathRoot, error) {
	if path == "" {
		return PathRoot{}, errors.New("root path cannot be empty")
	}

	return PathRoot{path: path}, nil
}

func NewOptionalPathRoot(path string) (*PathRoot, error) {
	if path == "" {
		return nil, nil
	}
	pathRoot, err := NewPathRoot(path)
	if err != nil {
		return nil, err
	}
	return &pathRoot, nil
}

func (pr PathRoot) Join(untrustedPath string) (string, error) {
	return securejoin.SecureJoin(pr.path, untrustedPath)
}

func (pr PathRoot) PathWithoutProtection() string {
	return pr.path
}

func (pr PathRoot) Format(f fmt.State, verb rune) {
	fmt.Fprintf(f, fmt.FormatString(f, verb), pr.path)
}

func (pr PathRoot) MarshalJSON() ([]byte, error) {
	return json.Marshal(pr.path)
}

func (pr *PathRoot) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &pr.path)
}

// Utilities for incremental migration.

func NewPathRootDeferred(path string) (PathRoot, error) {
	return NewPathRoot(path)
}

func NewOptionalPathRootDeferred(path string) (*PathRoot, error) {
	return NewOptionalPathRoot(path)
}

func (pr PathRoot) PathWithoutProtectionDeferred() string {
	return pr.PathWithoutProtection()
}

func NewPathRootTodo(path string) (PathRoot, error) {
	return NewPathRoot(path)
}

func NewOptionalPathRootTodo(path string) (*PathRoot, error) {
	return NewOptionalPathRoot(path)
}

func (pr PathRoot) PathWithoutProtectionTodo() string {
	return pr.PathWithoutProtection()
}
