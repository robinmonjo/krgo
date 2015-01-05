package dlrootfs

import (
	"bufio"
	"os"
	"strings"

	"github.com/docker/docker/pkg/archive"
)

func truncateID(id string) string {
	shortLen := 12
	if len(id) < shortLen {
		shortLen = len(id)
	}
	return id[:shortLen]
}

func ParseCredentials(credentials string) (string, string) {
	credentialsSplit := strings.SplitN(credentials, ":", 2)
	if len(credentialsSplit) != 2 {
		return "", ""
	}
	return credentialsSplit[0], credentialsSplit[1]
}

func ParseImageNameTag(imageNameTag string) (imageName string, imageTag string) {
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

//used mostly for debugging
func WriteArchiveToFile(archive archive.Archive, dest string) error {
	reader := bufio.NewReader(archive)
	tar, err := os.Create(dest)
	defer tar.Close()

	_, err = reader.WriteTo(tar)
	return err
}
