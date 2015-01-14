package main

import (
	"os"
	"strings"
)

//credentials format: <username>:<password>
func ParseCredentials(credentials string) (string, string) {
	credentialsSplit := strings.SplitN(credentials, ":", 2)
	if len(credentialsSplit) != 2 {
		return "", ""
	}
	return credentialsSplit[0], credentialsSplit[1]
}

//image format: <repository>/<image_name>:<tag>. tag defaults to latest, repository defaults to library
func ParseImageNameTag(imageNameTag string) (imageName, imageTag string) {
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

//return the index of string e in slice
func indexOf(slice []string, e string) int {
	for i, v := range slice {
		if v == e {
			return i
		}
	}
	return -1
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
