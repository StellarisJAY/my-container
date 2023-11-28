package main

import (
	"flag"
	"fmt"
	"github.com/StellarisJAY/my-container/container"
	"github.com/StellarisJAY/my-container/image"
	"github.com/StellarisJAY/my-container/util"
	"log"
	"os"
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
	fs.StringVar(&opts.Mount, "mount", "", "Mount points")
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
		container.ExecCommand(containerId, opts, fs.Args())
	case "exec":
		_ = fs.Parse(os.Args[2:])
		util.Must(container.ExecInContainer(containerId, fs.Args()), "Unable to exec in container ")
	case "ps":
		containers, err := container.GetRunningContainers()
		if err != nil {
			log.Fatalln("Unable to list running containers: ", err)
			return
		}
		fmt.Printf("%16s\t%8s\t%32s\n", "Container", "Pid", "Image")
		for _, c := range containers {
			fmt.Printf("%16s\t%8s\t%32s\n", c.ContainerId, c.Pid, c.Image)
		}
	case "images":
		fmt.Printf("%16s\t%8s\t%12s\n", "Name", "Tag", "Hash")
		if err := image.ListImages(); err != nil {
			log.Fatalln(err)
		}
	case "pull":
		_ = fs.Parse(os.Args[2:])
		_ = image.DownloadImageIfNotExist(imageName)
	}
}
