package main

import (
	"flag"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/container"
	"github.com/StellarisJAY/my-container/image"
	"github.com/StellarisJAY/my-container/util"
	"log"
	"os"
	"path"
)

func main() {
	if len(os.Args) == 1 {
		log.Fatalln("Invalid amount of arguments")
		return
	}
	cmd := os.Args[1]
	opts := &container.Options{}
	var (
		containerId string
		imageName   string
	)
	if os.Getuid() != 0 {
		log.Fatalln("Must run this program with root privilege")
		return
	}
	fs := flag.FlagSet{}
	fs.Float64Var(&opts.CpuLimit, "cpu", 1, "Set cpu limit")
	fs.IntVar(&opts.MemLimit, "mem", 1<<20, "Set memory limit")
	fs.StringVar(&containerId, "container", "", "Container id")
	fs.StringVar(&imageName, "image", "", "Image full name")
	switch cmd {
	case "run":
		_ = fs.Parse(os.Args[2:])
		imageHash := image.DownloadImageIfNotExist(imageName)
		log.Println("Image Hash: ", imageHash)
		containerId := container.CreateContainer(imageHash)
		log.Println("Container ID: ", containerId)
		container.Run(opts, containerId, os.Args[2:])
	case "child-mode":
		_ = fs.Parse(os.Args[2:])
		if len(fs.Args()) == 0 {
			log.Fatalln("Must provide container exec command")
			return
		}
		container.Exec(containerId, opts.CpuLimit, opts.MemLimit, fs.Args())
	case "exec":
		_ = fs.Parse(os.Args[2:])
		// 判断容器是否存在
		containerDir := path.Join(common.ContainerBaseDir, containerId)
		if _, err := os.Stat(containerDir); os.IsNotExist(err) {
			log.Println("Container doesn't exist ", containerId)
			return
		}
		// 挂载容器的文件系统layers
		util.Must(container.MountExistingContainerFS(containerId), "Unable to mount existing container fs")
		container.Run(opts, containerId, os.Args[2:])
	case "ps":
		containers, err := container.GetRunningContainers()
		if err != nil {
			log.Fatalln("Unable to list running containers: ", err)
			return
		}
		for _, c := range containers {
			fmt.Println(c)
		}
	case "images":
		if err := image.ListImages(); err != nil {
			log.Fatalln(err)
		}
	}
}
