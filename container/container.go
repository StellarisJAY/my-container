package container

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/image"
	"github.com/StellarisJAY/my-container/util"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"path"
	"strings"
)

func NewContainerId() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x",
		bytes[0], bytes[1],
		bytes[2], bytes[3],
		bytes[4], bytes[5],
		bytes[6], bytes[7])
}

func GetRunningContainers() ([]string, error) {
	dir := "/sys/fs/cgroup/cpu/my_container"
	if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		var containers []string
		for _, entry := range entries {
			if entry.IsDir() {
				containers = append(containers, entry.Name())
			}
		}
		return containers, nil
	} else {
		return nil, err
	}
}

// CreateContainer 从一个镜像创建容器，返回容器ID
func CreateContainer(imageHash string) string {
	containerId := NewContainerId()
	// 创建容器目录
	containerDirs := []string{
		path.Join(common.ContainerBaseDir, containerId, "fs", "mnt"),
		path.Join(common.ContainerBaseDir, containerId, "fs", "upperdir"),
		path.Join(common.ContainerBaseDir, containerId, "fs", "workdir"),
		path.Join(common.ContainerBaseDir, containerId, "fs", "layers"),
	}
	util.Must(util.CreateDirsIfNotExist(containerDirs), "Unable to make container dirs")
	// 挂载容器文件系统
	util.Must(createContainerFS(imageHash, containerId), "Unable to mount image layers ")
	return containerId
}

func MountExistingContainerFS(containerId string) error {
	containerFS := path.Join(common.ContainerBaseDir, containerId, "fs")
	var layers []string
	if stat, err := os.Stat(path.Join(containerFS, "layers")); os.IsNotExist(err) || !stat.IsDir() {
		return errors.New("container layers directory doesn't exist")
	}
	layerPath := path.Join(containerFS, "layers")
	entries, err := os.ReadDir(layerPath)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return errors.New("invalid container layers")
	}
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 16 {
			layers = append(layers, path.Join(layerPath, entry.Name()))
		}
	}
	return mountContainerLayers(containerId, layers)
}

func createContainerFS(imageHash string, containerId string) error {
	manifest, err := image.ParseManifest(imageHash)
	if err != nil {
		return err
	}
	imagePath := common.ImageBaseDir + imageHash
	containerFS := path.Join(common.ContainerBaseDir, containerId, "fs")

	layers := manifest[0].Layers
	var layerPaths []string
	for _, layer := range layers {
		layerPath := path.Join(containerFS, "layers", strings.TrimSuffix(layer, ".tar.gz")[:16])
		layerPaths = append(layerPaths, layerPath)
		log.Println("Untar layer: ", layer)
		// {image}/{layer}.tar.gz 解压到 {container}/fs/{i}/
		if err := util.Untar(path.Join(imagePath, layer), layerPath); err != nil {
			return err
		}
	}
	return mountContainerLayers(containerId, layerPaths)
}

func mountContainerLayers(containerId string, layers []string) error {
	containerFS := path.Join(common.ContainerBaseDir, containerId, "fs")
	mntPath := path.Join(containerFS, "mnt")
	// lowerdir为镜像的多个layers

	mntOptions := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		strings.Join(layers, ":"),
		path.Join(containerFS, "upperdir"),
		path.Join(containerFS, "workdir"))
	if err := unix.Mount("none", mntPath, "overlay", 0, mntOptions); err != nil {
		return fmt.Errorf("mount error %w", err)
	}
	return nil
}

func UmountContainerFS(containerId string) error {
	return unix.Unmount(path.Join(common.ContainerBaseDir, containerId, "fs", "mnt"), 0)
}
