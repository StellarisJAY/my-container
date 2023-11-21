package container

import (
	"github.com/StellarisJAY/my-container/cgroup"
	"github.com/StellarisJAY/my-container/util"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

type Options struct {
	CpuLimit float64
	MemLimit int
}

// Run 从image创建一个容器运行
func Run(opt *Options, imageHash string, args []string) {
	cmdArgs := []string{"child-mode"}
	cmdArgs = append(cmdArgs, opt.ToString()...)
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "-container", NewContainerId())
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
	util.Must(cmd.Run(), "namespace run failed")
	log.Println("container done")
}

// Exec 在一个容器中执行命令，该函数在child-mode子进程中进行，此时进程已经处于新的Namespace
func Exec(containerId string, cpuLimit float64, memLimit int, args []string) {
	// 创建CGroup控制CPU和内存配额
	cgroup.CreateCGroups(containerId)
	cgroup.ConfigureCGroup(containerId, cpuLimit, memLimit)
}

func (opt *Options) ToString() []string {
	return []string{
		"-cpu", strconv.FormatFloat(opt.CpuLimit, 'G', 2, 64),
		"-mem", strconv.Itoa(opt.MemLimit),
	}
}
