package cgroup

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func cpuTask() {
	var sum int64 = 0
	for {
		sum += 1
	}
}

func TestCpuLimit(t *testing.T) {
	var limit float64 = 2.5
	containerId := fmt.Sprintf("temp%d", time.Now().UnixMilli())
	CreateCGroups(containerId)
	ConfigureCGroup(containerId, limit, 1000)
	for i := 0; i < int(math.Ceil(limit)); i++ {
		go cpuTask()
	}
	select {}
}
