//go:build !windows

package daemon

import "os"

func makeTestDirectoryLink(target, link string) error {
	return os.Symlink(target, link)
}

func moveOpenedTestDirectory(source, destination string) (bool, error) {
	err := os.Rename(source, destination)
	return err == nil, err
}
