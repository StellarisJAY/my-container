package image

import (
	"encoding/json"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/util"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"log"
	"os"
	"strconv"
	"strings"
)

type imageEntries map[string]string

type Manifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

func getImageNameAndTag(src string) (string, string) {
	parts := strings.Split(src, ":")
	if len(parts) == 1 {
		return parts[0], "latest"
	} else {
		return parts[0], parts[1]
	}
}

func checkImageExistByName(name, tag string) (bool, string) {
	hash, err := getImageHash(name, tag)
	if err != nil {
		log.Fatalln(err)
		return false, ""
	}
	ok := hash != ""
	return ok, hash
}

func checkImageExistByHash(hashHex string) (bool, string, string) {
	nameAndTag, err := getImageNameAndTagByHash(hashHex)
	if err != nil {
		log.Fatalln(err)
		return false, "", ""
	}
	if nameAndTag == nil {
		return false, "", ""
	}
	return true, nameAndTag[0], nameAndTag[1]
}

func storeImageMetadata(name, tag, hashHex string) {
	if err := storeImage(name, tag, hashHex); err != nil {
		log.Fatalln(err)
	}
}

func downloadImageFile(image v1.Image, fullName, hashHex string) {
	saveDir := common.TempDir
	_ = util.CreateDirsIfNotExist([]string{saveDir})
	if err := crane.Save(image, fullName, saveDir+hashHex+".tar"); err != nil {
		log.Fatalln(err)
		return
	}
}

func untarImage(imageHash string) {
	imageTarPath := common.TempDir + imageHash + ".tar"
	targetPath := common.ImageBaseDir + imageHash
	if err := util.Untar(imageTarPath, targetPath); err != nil {
		log.Fatalln(fmt.Errorf("unable to untar image %w", err))
		return
	}
}

func ParseManifest(imageHash string) ([]Manifest, error) {
	manifestPath := common.ImageBaseDir + imageHash + "/manifest.json"
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read manifest.json %w", err)
	}
	var m []Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unable to parse manifest.json %w", err)
	}
	return m, nil
}

func DownloadImageIfNotExist(src string) string {
	name, tag := getImageNameAndTag(src)
	if ok, imageHash := checkImageExistByName(name, tag); !ok {
		log.Printf("Pulling image metadata for %s:%s", name, tag)
		fullName := strings.Join([]string{name, tag}, ":")
		image, err := crane.Pull(fullName)
		if err != nil {
			log.Fatal(err)
		}
		digest, _ := image.Digest()
		imageHashHex := digest.Hex[:12]
		if exist, altName, altTag := checkImageExistByHash(imageHashHex); exist {
			log.Printf("Required image %s:%s is the same as %s:%s, skip download", name, tag, altName, altTag)
			storeImageMetadata(name, tag, imageHashHex)
			return imageHashHex
		}
		storeImageMetadata(name, tag, imageHashHex)
		log.Println("Downloading image...")
		downloadImageFile(image, fullName, imageHashHex)
		untarImage(imageHashHex)
		return imageHashHex
	} else {
		log.Println("Image already exists. Skip download.")
		return imageHash
	}
}

func formatSize(size int) string {
	switch {
	case size < 1<<10:
		return strconv.Itoa(size) + "B"
	case size < 1<<20:
		return strconv.Itoa(size>>10) + "KiB"
	case size < 1<<30:
		return strconv.Itoa(size>>20) + "MiB"
	default:
		return strconv.Itoa(size>>30) + "GiB"
	}
}
