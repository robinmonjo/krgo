package dlrootfs

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/docker/docker/image"
	"github.com/docker/docker/utils"
)

func extractImageNameTag(imageNameTag string) (imageName, imageTag string) {
	if strings.Contains(imageNameTag, ":") {
		imageName = strings.Split(imageNameTag, ":")[0]
		imageTag = strings.Split(imageNameTag, ":")[1]
	} else {
		imageName = imageNameTag
		imageTag = "latest"
	}
	return
}

// directly inspired by LoadImage in the docker image package
func LoadImageFromJson(jsonPath string) (*image.Image, error) {
	// Open the JSON file to decode by streaming
	jsonSource, err := os.Open(jsonPath)
	if err != nil {
		return nil, err
	}
	defer jsonSource.Close()

	img := &image.Image{}
	dec := json.NewDecoder(jsonSource)

	// Decode the JSON data
	if err := dec.Decode(img); err != nil {
		return nil, err
	}
	if err := utils.ValidateID(img.ID); err != nil {
		return nil, err
	}
	img.Size = -1
	return img, nil
}
