package main

import (
	"io"
	"log"
	"os"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

const (
	IMAGE_ID    string = "6c3df001ea12dcf848ff51930954e2129ac8f5717ce98819237d2d5d3e8ddd25"
	REPOSITORY  string = "ubuntu"
	ROOTFS_DEST string = "./rootfs"
)

func assertErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	//resolving docker hub endpoint
	hostname, _, err := registry.ResolveRepositoryName(REPOSITORY)
	assertErr(err)

	registryEndpoint, err := registry.NewEndpoint(hostname)
	assertErr(err)

	//opening a session
	//empty auth config (probably used only for private repository or private images I guess)
	authConfig := &registry.AuthConfig{}
	var metaHeaders map[string][]string

	session, err := registry.NewSession(authConfig, registry.HTTPRequestFactory(metaHeaders), registryEndpoint, true)
	assertErr(err)

	//Get back token and endpoint for the repository
	data, err := session.GetRepositoryData(REPOSITORY)
	/*for k, v := range data.ImgList {
		log.Println("key = ", k, " value = ", v.Tag, " - ", v.ID)
	}*/

	assertErr(err)

	tokens := data.Tokens
	repoEndpoint := data.Endpoints[0]

	//Download image history (get back all the layers)
	history, err := session.GetRemoteHistory(IMAGE_ID, repoEndpoint, tokens)
	assertErr(err)

	log.Println("Image", IMAGE_ID, "is made of", len(history), "layers:", history)

	os.MkdirAll(ROOTFS_DEST, 0777)

	for i := len(history) - 1; i >= 0; i-- {
		imageId := history[i]

		log.Println("Downloading layer", imageId, "...")
		layerData, err := downloadImageLayer(session, imageId, repoEndpoint, tokens)
		defer layerData.Close()
		assertErr(err)

		log.Println("Untaring layer", imageId)
		err = archive.Untar(layerData, ROOTFS_DEST, nil)
		assertErr(err)

		log.Println("done", imageId)
	}

	log.Println("All good")

}

func downloadImageLayer(session *registry.Session, imageId, endpoint string, tokens []string) (io.ReadCloser, error) {
	//Get back image information
	_, imgSize, err := session.GetRemoteImageJSON(imageId, endpoint, tokens)
	if err != nil {
		return nil, err
	}
	return session.GetRemoteImageLayer(imageId, endpoint, tokens, int64(imgSize))
}
