package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/utils"
)

//credentials format: <username>:<password>
func parseCredentials(credentials string) (string, string) {
	credentialsSplit := strings.SplitN(credentials, ":", 2)
	if len(credentialsSplit) != 2 {
		return "", ""
	}
	return credentialsSplit[0], credentialsSplit[1]
}

//image format: <repository>/<image_name>:<tag>. tag defaults to latest, repository defaults to library
func parseImageNameTag(imageNameTag string) (imageName, imageTag string) {
	if strings.Contains(imageNameTag, ":") {
		imageName = strings.SplitN(imageNameTag, ":", 2)[0]
		imageTag = strings.SplitN(imageNameTag, ":", 2)[1]
	} else {
		imageName = imageNameTag
		imageTag = "latest"
	}

	if !strings.Contains(imageName, "/") {
		imageName = "library/" + imageName
	}
	return
}

//all branches in git repo must be formatted thiw way: layer_N_imageId
func validateBranch(branch string) error {
	errToReturn := fmt.Errorf("invalide date format: %v expecting layer_n_imageid", branch)
	comps := strings.Split(branch, "_")
	if len(comps) != 3 {
		return errToReturn
	}
	if comps[0] != "layer" {
		return errToReturn
	}
	if _, err := strconv.ParseUint(comps[1], 10, 32); err != nil {
		return errToReturn
	}
	if err := utils.ValidateID(comps[2]); err != nil {
		return errToReturn
	}
	return nil
}

func exportLayerNumberFromBranch(branch string) (int64, error) {
	if err := validateBranch(branch); err != nil {
		return -1, err
	}
	i, _ := strconv.ParseInt(strings.Split(branch, "_")[1], 10, 64)
	return i, nil
}

//fileExists reports whether the named file or directory exists
func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
