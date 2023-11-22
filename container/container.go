package container

import (
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/image"
	"github.com/StellarisJAY/my-container/util"
	"golang.org/x/sys/unix"
	"log"
	"path"
	"strings"
	"time"
)

func NewContainerId() string {
	// todo container ID
	return fmt.Sprintf("%d", time.Now().UnixMilli())
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
	util.Must(mountContainerFS(imageHash, containerId), "Unable to mount image layers ")
	return containerId
}

func mountContainerFS(imageHash string, containerId string) error {
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

	mntPath := path.Join(containerFS, "mnt")
	// lowerdir为镜像的多个layers
	mntOptions := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		strings.Join(layerPaths, ":"),
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
