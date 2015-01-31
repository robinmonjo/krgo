package main

import (
	"os"
	"strings"
)

//credentials format: <username>:<password>
func parseCredentials(credentials string) (string, string) {
	comps := strings.SplitN(credentials, ":", 2)
	if len(comps) != 2 {
		return "", ""
	}
	return comps[0], comps[1]
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

//fileExists reports whether the named file or directory exists
func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
