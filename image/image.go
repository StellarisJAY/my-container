package image

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/StellarisJAY/my-container/common"
	"github.com/StellarisJAY/my-container/util"
	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"io/ioutil"
	"log"
	"os"
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

func readImagesMetadata() map[string]imageEntries {
	path := common.ImageBaseDir + "images.json"
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		log.Fatalln("Unable to read images.json: ", err)
		return nil
	}
	result := make(map[string]imageEntries)
	if err := json.Unmarshal(bytes, &result); err != nil {
		log.Fatalf("Invalid images.json data ")
		return nil
	}
	return result
}

func writeImagesMetadata(metadata map[string]imageEntries) {
	path := common.ImageBaseDir + "images.json"
	if err := util.CreateFileIfNotExist(path); err != nil {
		log.Fatalln(err)
		return
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		log.Fatalln("Unable to marshal metadata to json ", err)
		return
	}
	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		log.Fatalln("Unable to write images.json ", err)
	}
}

func checkImageExistByName(name, tag string) (bool, string) {
	metadata := readImagesMetadata()
	if image, ok := metadata[name]; ok {
		if h, ok := image[tag]; ok {
			return true, h
		}
	}
	return false, ""
}

func checkImageExistByHash(hashHex string) (bool, string, string) {
	metadata := readImagesMetadata()
	for name, v := range metadata {
		for tag, h := range v {
			if h == hashHex {
				return true, name, tag
			}
		}
	}
	return false, "", ""
}

func storeImageMetadata(name, tag, hashHex string) {
	var metadata map[string]imageEntries
	if m := readImagesMetadata(); m != nil {
		metadata = m
	} else {
		metadata = make(map[string]imageEntries)
	}
	var entry imageEntries
	if m, ok := metadata[name]; !ok {
		entry = make(map[string]string)
	} else {
		entry = m
	}
	entry[tag] = hashHex
	metadata[name] = entry
	writeImagesMetadata(metadata)
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
		log.Printf("Downloading image metadata for %s:%s", name, tag)
		fullName := strings.Join([]string{name, tag}, ":")
		image, err := crane.Pull(fullName)
		if err != nil {
			log.Fatal(err)
		}
		digest, _ := image.Digest()
		imageHashHex := digest.Hex[:12]
		log.Println("Image Hash Hex: ", imageHashHex)
		if exist, altName, altTag := checkImageExistByHash(imageHashHex); exist {
			log.Printf("Required image %s:%s is the same as %s:%s, skip download", name, tag, altName, altTag)
			storeImageMetadata(name, tag, imageHashHex)
			return imageHashHex
		}
		storeImageMetadata(name, tag, imageHashHex)
		downloadImageFile(image, fullName, imageHashHex)
		untarImage(imageHashHex)
		return imageHashHex
	} else {
		log.Println("Image already exists. Skip download.")
		return imageHash
	}
}
