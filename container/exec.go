package container

import (
	"errors"
	"github.com/StellarisJAY/my-container/cgroup"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/util"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"path"
)

func ExecInContainer(containerId string, args []string) error {
	pid, err := getRunningContainerPid(containerId)
	if err != nil {
		return err
	}
	// 容器namespace文件fd
	ns := path.Join("/proc", pid, "ns")
	mntFd, mntErr := os.Open(path.Join(ns, "mnt"))
	ipcFd, ipcErr := os.Open(path.Join(ns, "ipc"))
	pidFd, pidErr := os.Open(path.Join(ns, "pid"))
	netFd, netErr := os.Open(path.Join(ns, "net"))
	utsFd, utsErr := os.Open(path.Join(ns, "uts"))
	if mntErr != nil || ipcErr != nil || pidErr != nil || netErr != nil || utsErr != nil {
		return errors.New("can't open namespace files")
	}
	// 设置当前进程到容器namespace
	unix.Setns(int(ipcFd.Fd()), unix.CLONE_NEWIPC)
	unix.Setns(int(mntFd.Fd()), unix.CLONE_NEWNS)
	unix.Setns(int(pidFd.Fd()), unix.CLONE_NEWPID)
	unix.Setns(int(netFd.Fd()), unix.CLONE_NEWNET)
	unix.Setns(int(utsFd.Fd()), unix.CLONE_NEWUTS)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// exec的进程加入容器的cgroup
	cgroup.CreateCGroups(containerId)
	// 转到容器root目录，创建子进程执行命令
	mntPath := path.Join(common.ContainerBaseDir, containerId, "fs", "mnt")
	util.Must(unix.Chroot(mntPath), "Unable to chroot to mnt path")
	util.Must(unix.Chdir("/"), "Unable to chdir to root dir")
	util.Must(cmd.Run(), "Unable to exec command")
	return nil
}
