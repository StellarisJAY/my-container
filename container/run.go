package container

import (
	"errors"
	"github.com/StellarisJAY/my-container/cgroup"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/network"
	"github.com/StellarisJAY/my-container/util"
	"github.com/StellarisJAY/my-container/volume"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
)

type Options struct {
	CpuLimit float64
	MemLimit int
	Mount    string
	Volume   string
}

// Run 从image创建一个容器运行
func Run(opt *Options, containerId string, args []string) {
	// 创建宿主机网桥
	util.Must(network.SetupBridge(), "Unable to set up bridge")
	network.InitIptables()
	// 宿主机与网桥的veth
	util.Must(network.SetupHostVeth(), "Unable to connect host veth to bridge")
	// 创建容器网络命名空间
	util.Must(network.CreateNetworkNamespace(containerId), "Unable to create network namespace")
	util.Must(network.CreateVeth(containerId), "Unable to create container veth")
	util.Must(network.SetupVethToBridge(containerId), "Unable to setup container veth to bridge ")
	prepareVethInNamespace(containerId)

	cmdArgs := []string{"child-mode"}
	cmdArgs = append(cmdArgs, opt.ToString()...)
	cmdArgs = append(cmdArgs, "-container", containerId)
	cmdArgs = append(cmdArgs, args...)
	// cmd.Run 以child-mode参数创建子进程并运行my_container
	cmd := exec.Command("/proc/self/exe", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	// 设置子进程的Namespace, 子进程PID将为自己Namespace的1
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID |
			syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
	}
	originalNS, _ := unix.Open("/proc/self/ns/net", unix.O_RDONLY, 0644)
	log.Println("Cmd Args: ", cmd.Args)
	// 进入子进程
	util.Must(cmd.Run(), "namespace run failed")
	// 回到父进程
	util.Must(unix.Setns(originalNS, unix.CLONE_NEWNET), "Unable to switch back to host netns")
	util.Must(UmountContainerFS(containerId), "Unable to unmount container fs")
	network.UnmountNetworkNamespace(containerId)
	util.Must(cgroup.RemoveCGroups(containerId), "Unable to remove cgroups ")
	util.Must(os.RemoveAll(path.Join(common.ContainerBaseDir, containerId)), "Unable to remove container dir")
	log.Println("container done")
}

// ExecCommand 在一个容器中执行命令，该函数在child-mode子进程中进行，此时进程已经处于新的Namespace
func ExecCommand(containerId string, options *Options, args []string) {
	// 创建CGroup控制CPU和内存配额
	cgroup.CreateCGroups(containerId)
	cgroup.ConfigureCGroup(containerId, options.CpuLimit, options.MemLimit)

	// bind mounts
	var bindMntPoint string
	if m, err := bindMounts(containerId, options.Mount); err != nil {
		log.Println("Unable to mount host directory ", err)
	} else {
		bindMntPoint = m
	}
	// mount volume
	var volumeMntPoint string
	if v, err := mountVolume(containerId, options.Volume); err != nil {
		log.Fatalln(err)
		return
	} else {
		volumeMntPoint = v
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	util.Must(unix.Sethostname([]byte(containerId)), "Unable to set container host name")
	util.Must(network.JoinNetworkNamespace(containerId), "Unable to switch to container netns")
	network.SetupLocalhostInterface()
	// 将当前namespace的根目录设置到容器根目录
	mntPath := path.Join(common.ContainerBaseDir, containerId, "fs", "mnt")
	util.Must(unix.Chroot(mntPath), "Unable to chroot to container file system")
	util.Must(unix.Chdir("/"), "Unable to chdir to container root")
	_ = util.CreateDirsIfNotExist([]string{"/proc", "/sys"})
	// 挂载/proc /sys
	util.Must(unix.Mount("proc", "/proc", "proc", 0, ""), "Unable to mount /proc")
	util.Must(unix.Mount("sysfs", "/sys", "sysfs", 0, ""), "Unable to mount /sys")

	_ = cmd.Run()
	network.RemoveVeth(containerId, "-ns")
	if bindMntPoint != "" {
		unix.Unmount(bindMntPoint, 0)
	}
	if volumeMntPoint != "" {
		unix.Unmount(volumeMntPoint, 0)
	}
	util.Must(unix.Unmount("/proc", 0), "Unable to unmount /proc")
	util.Must(unix.Unmount("/sys", 0), "Unable to unmount /sys")
}

func bindMounts(containerId string, mntOptions string) (string, error) {
	parts := strings.Split(mntOptions, ",")
	var src, dest string
	for _, option := range parts {
		kv := strings.SplitN(option, "=", 2)
		if len(kv) != 2 {
			return "", errors.New("invalid mnt options")
		}
		key, value := kv[0], kv[1]
		switch key {
		case "src":
			src = value
		case "dest":
			dest = value
		default:
			continue
		}
	}
	if src == "" || dest == "" {
		return "", errors.New("invalid mount source or destination")
	}
	log.Println("Mount host directory, src: ", src, ", dest: ", dest)
	mntPoint := path.Join(common.ContainerBaseDir, containerId, "fs", "mnt", dest)
	_ = util.CreateDirsIfNotExist([]string{mntPoint})
	// 使用bind mount将宿主机目录挂载到容器目录
	return dest, unix.Mount(src, mntPoint, "", unix.MS_BIND, "")
}

func mountVolume(containerId string, volumeOption string) (string, error) {
	parts := strings.SplitN(volumeOption, ":", 2)
	volumeName, dest := parts[0], parts[1]
	v, err := volume.InspectVolume(volumeName)
	if err != nil {
		return "", err
	}
	hostPath := v.MountPoint
	containerPath := path.Join(common.ContainerBaseDir, containerId, "fs", "mnt", dest)
	return dest, unix.Mount(hostPath, containerPath, "", unix.MS_BIND, "")
}

func prepareVethInNamespace(containerId string) {
	cmd := exec.Command("/proc/self/exe", "setup-veth", "-container", containerId)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func (opt *Options) ToString() []string {
	return []string{
		"-cpu", strconv.FormatFloat(opt.CpuLimit, 'G', 2, 64),
		"-mem", strconv.Itoa(opt.MemLimit),
	}
}
