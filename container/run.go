package container

import (
	"errors"
	"github.com/StellarisJAY/my-container/cgroup"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/util"
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
}

// Run 从image创建一个容器运行
func Run(opt *Options, containerId string, args []string) {
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
	log.Println("Cmd Args: ", cmd.Args)
	// 进入子进程
	util.Must(cmd.Run(), "namespace run failed")
	// 回到父进程
	util.Must(UmountContainerFS(containerId), "Unable to unmount container fs")
	util.Must(cgroup.RemoveCGroups(containerId), "Unable to remove cgroups ")
	util.Must(os.RemoveAll(path.Join(common.ContainerBaseDir, containerId)), "Unable to remove container dir")
	log.Println("container done")
}

// ExecCommand 在一个容器中执行命令，该函数在child-mode子进程中进行，此时进程已经处于新的Namespace
func ExecCommand(containerId string, options *Options, args []string) {
	// 创建CGroup控制CPU和内存配额
	cgroup.CreateCGroups(containerId)
	cgroup.ConfigureCGroup(containerId, options.CpuLimit, options.MemLimit)

	mntPath := path.Join(common.ContainerBaseDir, containerId, "fs", "mnt")
	var hostDirMntPoint string
	if mntPoint, err := bindMounts(containerId, options.Mount); err != nil {
		log.Println("Unable to mount host directory ", err)
	} else {
		hostDirMntPoint = mntPoint
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	util.Must(unix.Sethostname([]byte(containerId)), "Unable to set container host name")
	// 将当前namespace的根目录设置到容器根目录
	util.Must(unix.Chroot(mntPath), "Unable to chroot to container file system")
	util.Must(unix.Chdir("/"), "Unable to chdir to container root")
	util.CreateDirsIfNotExist([]string{"/proc", "/sys"})
	// 挂载/proc /sys
	util.Must(unix.Mount("proc", "/proc", "proc", 0, ""), "Unable to mount /proc")
	util.Must(unix.Mount("sysfs", "/sys", "sysfs", 0, ""), "Unable to mount /sys")

	_ = cmd.Run()
	// unmount 宿主机目录
	_ = unix.Unmount(hostDirMntPoint, 0)
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

func (opt *Options) ToString() []string {
	return []string{
		"-cpu", strconv.FormatFloat(opt.CpuLimit, 'G', 2, 64),
		"-mem", strconv.Itoa(opt.MemLimit),
	}
}
