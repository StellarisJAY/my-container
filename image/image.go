package image

import (
	"encoding/json"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/config"
	"github.com/StellarisJAY/my-container/util"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"log"
	"os"
	"path"
	"strings"
)

type imageEntries map[string]string

type Manifest struct {
	Config   string   `json:"Config"`
	RepoTags []string `json:"RepoTags"`
	Layers   []string `json:"Layers"`
}

func pullImageFromCustomSource(src string) (v1.Image, error) {
	ref, err := name.ParseReference(src)
	if err != nil {
		return nil, err
	}
	image, err := remote.Image(ref, remote.WithJobs(1))
	if err != nil {
		return nil, err
	}
	return image, nil
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

func downloadImageFile(image v1.Image, src, hashHex string) error {
	saveDir := common.TempDir
	_ = util.CreateDirsIfNotExist([]string{saveDir})
	return crane.Save(image, src, path.Join(saveDir, hashHex+".tar"))
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
	imageName, tag := getImageNameAndTag(src)
	if ok, imageHash := checkImageExistByName(imageName, tag); !ok {
		log.Printf("Pulling image metadata for %s:%s", imageName, tag)
		fullName := strings.Join([]string{imageName, tag}, ":")
		src := config.GlobalConfig.Registries[0] + "/library/" + fullName
		log.Println("Pulling image from ", src)
		image, err := pullImageFromCustomSource(src)
		if err != nil {
			log.Fatal(err)
			return ""
		}
		digest, _ := image.Digest()
		imageHashHex := digest.Hex[:12]
		if exist, altName, altTag := checkImageExistByHash(imageHashHex); exist {
			log.Printf("Required image %s:%s is the same as %s:%s, skip download", imageName, tag, altName, altTag)
			storeImageMetadata(imageName, tag, imageHashHex)
			return imageHashHex
		}
		storeImageMetadata(imageName, tag, imageHashHex)
		log.Println("Downloading image...")
		if err := downloadImageFile(image, src, imageHashHex); err != nil {
			log.Fatalln("Unable to download image ", err)
			return ""
		}
		untarImage(imageHashHex)
		return imageHashHex
	} else {
		log.Println("Image already exists. Skip download.")
		return imageHash
	}
}
