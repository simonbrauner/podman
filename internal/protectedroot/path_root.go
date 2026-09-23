package protectedroot

import securejoin "github.com/cyphar/filepath-securejoin"

type PathRoot struct {
	path string
}

func NewPathRoot(path string) *PathRoot {
	return &PathRoot{path: path}
}

func (pr *PathRoot) Join(untrustedPath string) (string, error) {
	return securejoin.SecureJoin(pr.path, untrustedPath)
}

func (pr *PathRoot) PathWithoutProtection() string {
	return pr.path
}
