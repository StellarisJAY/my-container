package container

import (
	"bufio"
	"bytes"
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

type RunningContainerInfo struct {
	ContainerId string
	Image       string
	Pid         string
}

func NewContainerId() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x",
		b[0], b[1],
		b[2], b[3],
		b[4], b[5],
		b[6], b[7])
}

func GetRunningContainers() ([]RunningContainerInfo, error) {
	dir := "/sys/fs/cgroup/cpu/my_container"
	if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		var containers []RunningContainerInfo
		for _, entry := range entries {
			if entry.IsDir() {
				info := getRunningContainerInfo(entry.Name())
				if info == nil {
					continue
				}
				containers = append(containers, *info)
			}
		}
		return containers, nil
	} else {
		return nil, err
	}
}

func getRunningContainerInfo(containerId string) *RunningContainerInfo {
	pid, err := getRunningContainerPid(containerId)
	if err != nil {
		log.Println("Unable to get pid for ", containerId, " error: ", err)
		return nil
	}
	imageHash, err := getContainerImage(containerId)
	if err != nil {
		log.Println("Unable to get image hash, error ", err)
		return nil
	}
	nameAndTag, err := image.GetImageNameAndTagByHash(imageHash)
	if err != nil {
		log.Println("Unable to get image name and tag, error: ", err)
		return nil
	}
	return &RunningContainerInfo{
		ContainerId: containerId,
		Pid:         pid,
		Image:       strings.Join(nameAndTag, ":"),
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
	layers := manifest[0].Layers
	var layerPaths []string
	for _, layer := range layers {
		layerPath := path.Join(imagePath, "layers", strings.TrimSuffix(layer, ".tar.gz")[:16])
		layerPaths = append(layerPaths, layerPath)
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

func getRunningContainerPid(containerId string) (string, error) {
	dir := "/sys/fs/cgroup/cpu/my_container/" + containerId
	if stat, err := os.Stat(dir); os.IsNotExist(err) || !stat.IsDir() {
		return "", os.ErrNotExist
	}
	data, err := os.ReadFile(path.Join(dir, "cgroup.procs"))
	if err != nil {
		return "", err
	}
	pids := strings.Split(string(data), "\n")
	return pids[1], nil
}

func getContainerImage(containerId string) (string, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("can't read /proc/mounts error: %w", err)
	}
	reader := bufio.NewReader(bytes.NewReader(data))
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		parts := strings.Split(line, " ")
		_, target, mountType, mountOptions := parts[0], parts[1], parts[2], parts[3]
		if mountType != "overlay" || !strings.Contains(target, containerId) {
			continue
		}
		options := strings.Split(mountOptions, ",")
		for _, opt := range options {
			if !strings.HasPrefix(opt, "lowerdir=") {
				continue
			}
			lowerDirs := strings.TrimPrefix(opt, "lowerdir=")
			layer0 := strings.TrimPrefix(lowerDirs, common.ImageBaseDir)
			return strings.Split(layer0, "/")[0], nil
		}
	}
	return "", fmt.Errorf("can't find mount points for container %s", containerId)
}
