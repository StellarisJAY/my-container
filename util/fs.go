package util

import (
	"errors"
	"fmt"
	"os"
	"strings"
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

func CreateFileIfNotExist(path string) error {
	dir := path[:strings.LastIndex(path, "/")]
	if err := CreateDirsIfNotExist([]string{dir}); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Create(path); err != nil {
			return fmt.Errorf("unable to create file %s, %w", path, err)
		}
	}
	return nil
}
