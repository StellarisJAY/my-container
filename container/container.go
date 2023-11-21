package container

import (
	"fmt"
	"time"
)

func NewContainerId() string {
	// todo container ID
	return fmt.Sprintf("%d", time.Now().UnixMilli())
}
