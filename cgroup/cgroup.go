package cgroup

import (
	"fmt"
	"github.com/StellarisJAY/my-container/util"
	"os"
	"runtime"
	"strconv"
)

func CreateCGroups(containerId string) {
	cgroupDirs := getCGroupDirs(containerId)
	util.Must(util.CreateDirsIfNotExist(cgroupDirs), "failed to create cgroup dirs")
	pid := os.Getpid()
	for _, dir := range cgroupDirs {
		util.Must(os.WriteFile(dir+"/notify_on_release", []byte{'1'}, 0700),
			"failed to write notify_on_release")
		// 当前进程添加到cgroup
		util.Must(os.WriteFile(dir+"/cgroup.procs", []byte(strconv.Itoa(pid)), 0700),
			"failed to write pid ")
	}
}

func RemoveCGroups(containerId string) error {
	cgroupDirs := getCGroupDirs(containerId)
	for _, dir := range cgroupDirs {
		if err := util.RemoveDir(dir); err != nil {
			return err
		}
	}
	return nil
}

func getCGroupDirs(containerId string) []string {
	return []string{
		"/sys/fs/cgroup/cpu/my_container/" + containerId,
		"/sys/fs/cgroup/memory/my_container/" + containerId,
		"/sys/fs/cgroup/pids/my_container/" + containerId,
	}
}

func ConfigureCGroup(containerId string, cpuLimit float64, memLimitMB int) {
	if cpuLimit > 0 {
		util.Must(setCPULimit(containerId, cpuLimit), "Unable to set cpu limit")
	}
	if memLimitMB > 0 {
		util.Must(setMemoryLimit(containerId, memLimitMB), "Unable to set memory limit")
	}
}

// setCPULimit 设置容器的CPU配比，limit为CPU个数
func setCPULimit(containerId string, limit float64) error {
	cfsPeriod := "/sys/fs/cgroup/cpu/my_container/" + containerId + "/cpu.cfs_period_us"
	cfsQuota := "/sys/fs/cgroup/cpu/my_container/" + containerId + "/cpu.cfs_quota_us"

	if limit > float64(runtime.NumCPU()) {
		return fmt.Errorf("cpu limit exceeded logical CPU NUM")
	}
	if err := os.WriteFile(cfsPeriod, []byte(strconv.Itoa(100000)), 0600); err != nil {
		return err
	}
	return os.WriteFile(cfsQuota,
		[]byte(strconv.Itoa(int(100000*limit))),
		0600)
}

// setMemoryLimit 设置容器内存限制和交换区大小
func setMemoryLimit(containerId string, limitMB int) error {
	limitInBytes := "/sys/fs/cgroup/memory/my_container/" + containerId + "/memory.limit_in_bytes"
	return os.WriteFile(limitInBytes, []byte(strconv.Itoa(limitMB*1024)), 0600)
}
