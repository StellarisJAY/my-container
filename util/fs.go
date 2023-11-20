package util

import (
	"errors"
	"fmt"
	"os"
)

func CreateDirsIfNotExist(dirs []string) error {
	for _, dir := range dirs {
		_, err := os.Stat(dir)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(dir, 0600); err != nil {
				return fmt.Errorf("mkdir %s error %w", dir, err)
			}
		}
	}
	return nil
}

func RemoveDir(dir string) error {
	return os.RemoveAll(dir)
}
