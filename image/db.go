package image

import (
	"errors"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/util"
	"github.com/boltdb/bolt"
	"path"
)

type Database struct{}

const (
	dbFile = common.ImageBaseDir + "image.db"
)

var errImageFound = errors.New("image found")

func init() {
	util.Must(util.CreateDirsIfNotExist([]string{path.Dir(dbFile)}), "Unable to create database dir")
}

func storeImage(name, tag, hash string) error {
	db, err := bolt.Open(dbFile, 0644, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return err
		}
		return b.Put([]byte(tag), []byte(hash))
	})
}

func getImageHash(name, tag string) (string, error) {
	db, err := bolt.Open(dbFile, 0644, nil)
	if err != nil {
		return "", err
	}
	defer db.Close()
	var hash string
	e := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(name))
		if b == nil {
			return nil
		}
		hash = string(b.Get([]byte(tag)))
		return nil
	})
	if e != nil {
		return "", e
	}
	return hash, nil
}

func getImageNameAndTagByHash(hash string) ([]string, error) {
	db, err := bolt.Open(dbFile, 0644, nil)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	result := make([]string, 2)
	e := db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(imageName []byte, b *bolt.Bucket) error {
			return b.ForEach(func(tag, v []byte) error {
				if string(v) == hash {
					result[0], result[1] = string(imageName), string(tag)
					return errImageFound
				}
				return nil
			})
		})
	})
	if errors.Is(e, errImageFound) {
		return result, nil
	} else {
		return nil, e
	}
}

func ListImages() error {
	db, err := bolt.Open(dbFile, 0644, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(imageName []byte, b *bolt.Bucket) error {
			return b.ForEach(func(tag, hash []byte) error {
				fmt.Printf("%16s\t%8s\t%12s\n", string(imageName), string(tag), string(hash))
				return nil
			})
		})
	})
}
