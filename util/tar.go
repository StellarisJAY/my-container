package util

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Untar(tarFile string, target string) error {
	// 创建目标目录
	if err := os.MkdirAll(target, 0644); err != nil {
		return err
	}
	file, err := os.OpenFile(tarFile, os.O_RDONLY, 0444)
	if err != nil {
		return err
	}
	defer file.Close()
	var reader *tar.Reader
	if strings.HasSuffix(tarFile, ".gz") {
		gr, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gr.Close()
		reader = tar.NewReader(gr)
	} else {
		reader = tar.NewReader(file)
	}

	hardLinks := make(map[string]string)
	for header, err := reader.Next(); err != io.EOF; header, err = reader.Next() {
		if err != nil {
			return err
		}
		fileName := filepath.Join(target, header.Name)
		info := header.FileInfo()
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(fileName); os.IsNotExist(err) {
				_ = os.MkdirAll(fileName, 0644)
			}
		case tar.TypeReg:
			if _, err := os.Stat(filepath.Dir(fileName)); os.IsNotExist(err) {
				_ = os.MkdirAll(filepath.Dir(fileName), 0644)
			}
			f, err := os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, info.Mode().Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, reader); err != nil {
				return err
			}
		case tar.TypeLink:
			// 硬链接目标文件可能还未解压，记录链接关系
			linkPath := filepath.Join(target, header.Linkname)
			hardLinks[fileName] = linkPath
		case tar.TypeSymlink:
			// 符号链接的linkName已经是绝对地址，不需要Join
			if err := os.Symlink(header.Linkname, fileName); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported file type : %c", header.Typeflag)
		}
	}
	// 创建硬链接
	for newName, oldName := range hardLinks {
		if err := os.Link(oldName, newName); err != nil {
			return err
		}
	}
	return nil
}
