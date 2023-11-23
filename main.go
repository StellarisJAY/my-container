package main

import (
	"flag"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/container"
	"github.com/StellarisJAY/my-container/image"
	"log"
	"os"
	"path"
)

func main() {
	cmd := os.Args[1]
	opts := &container.Options{}
	var (
		containerId string
		imageName   string
	)
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
		container.Exec(containerId, opts.CpuLimit, opts.MemLimit, fs.Args())
	case "exec":
		_ = fs.Parse(os.Args[2:])
		containerDir := path.Join(common.ContainerBaseDir, containerId)
		if _, err := os.Stat(containerDir); os.IsNotExist(err) {
			log.Println("Container doesn't exist ", containerId)
			return
		}
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
