package main

import (
	"fmt"
	"io"
	//"io/ioutil"

	"github.com/docker/docker/registry"
)

const (
	IMAGE_ID   string = "9942dd43ff211ba917d03637006a83934e847c003bef900e4808be8021dca7bd"
	REPOSITORY string = "ubuntu"
)

func assertErr(err error) {
	if err != nil {
		panic(err)
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
	assertErr(err)

	tokens := data.Tokens
	repoEndpoint := data.Endpoints[0]

	//Download image history (get back all the layers)
	history, err := session.GetRemoteHistory(IMAGE_ID, repoEndpoint, tokens)
	assertErr(err)

	fmt.Println("Image ", IMAGE_ID, " is made of ", len(history), " layers: ", history)

	layerData, err := downloadImageLayer(session, IMAGE_ID, repoEndpoint, tokens)
	defer layerData.Close()
	assertErr(err)

	fmt.Println("All good")

}

func downloadImageLayer(session *registry.Session, imageId, endpoint string, tokens []string) (io.ReadCloser, error) {
	//Get back image information
	_, imgSize, err := session.GetRemoteImageJSON(imageId, endpoint, tokens)
	if err != nil {
		return nil, err
	}
	return session.GetRemoteImageLayer(imageId, endpoint, tokens, int64(imgSize))
}
