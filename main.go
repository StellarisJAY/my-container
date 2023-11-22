package main

import (
	"flag"
	"github.com/StellarisJAY/my-container/container"
	"github.com/StellarisJAY/my-container/image"
	"log"
	"os"
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
		container.Run(opts, containerId, imageHash, os.Args[2:])
	case "child-mode":
		_ = fs.Parse(os.Args[2:])
		container.Exec(containerId, opts.CpuLimit, opts.MemLimit, os.Args[2:])
	}
}
