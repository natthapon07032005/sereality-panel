//go:build !windows

package database

import "os"

func replaceFileAtomicallyPlatform(sourcePath, destinationPath string) error {
	return os.Rename(sourcePath, destinationPath)
}
