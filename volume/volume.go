package volume

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/util"
	"github.com/boltdb/bolt"
	"path/filepath"
	"time"
)

type Volume struct {
	Name       string    `json:"Name"`
	CreatedAt  time.Time `json:"CreatedAt"`
	MountPoint string    `json:"MountPoint"`
}

var (
	ErrVolumeNotFound = errors.New("no such volume")
)

const (
	volumeDBPath = common.VolumeDir + "metadata.db"
)

func init() {
	util.Must(util.CreateDirsIfNotExist([]string{common.VolumeDir}), "Unable to create volume path")
}

func HandleCommand(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: my-container volume COMMAND args...")
		return
	}
	switch args[0] {
	case "create":
		handleCreateVolume(args)
	case "ls":
	case "inspect":
		handleInspectVolume(args)
	case "rm":
	default:
		fmt.Println("unsupported volume command: ", args[0])
	}
}

func randomVolumeName() string {
	return ""
}

func handleCreateVolume(args []string) {
	var name string
	if len(args) >= 2 {
		name = args[1]
	} else {
		name = randomVolumeName()
	}
	util.Must(CreateVolume(name), "Unable to create volume")
}

func handleInspectVolume(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: my-container volume inspect NAME")
		return
	}
	v, err := InspectVolume(args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println(v.ToString())
}

func CreateVolume(name string) error {
	db, err := bolt.Open(volumeDBPath, 0644, nil)
	if err != nil {
		return fmt.Errorf("unable to open database file %w", err)
	}
	defer db.Close()

	mountPoint := filepath.Join(common.VolumeDir, name, "_data")
	_ = util.CreateDirsIfNotExist([]string{mountPoint})
	data, _ := json.Marshal(&Volume{
		Name:       name,
		CreatedAt:  time.Now(),
		MountPoint: mountPoint,
	})
	return db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte("metadata"))
		return b.Put([]byte(name), data)
	})
}

func InspectVolume(name string) (*Volume, error) {
	db, err := bolt.Open(volumeDBPath, 0644, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to open database file %w", err)
	}
	defer db.Close()
	var v *Volume
	_ = db.View(func(tx *bolt.Tx) error {
		if b := tx.Bucket([]byte("metadata")); b != nil {
			data := b.Get([]byte(name))
			if data != nil {
				var res Volume
				_ = json.Unmarshal(data, &res)
				v = &res
			}
		}
		return nil
	})
	if v == nil {
		return nil, ErrVolumeNotFound
	}
	return v, nil
}

func (v *Volume) ToString() string {
	return fmt.Sprintf("CreatedAt: %s\nName: %s\nMountPoint: %s\n",
		v.CreatedAt.Format(time.DateTime),
		v.Name,
		v.MountPoint)
}
