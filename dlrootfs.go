package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

var rootfsDest *string = flag.String("d", "./rootfs", "destination of the resulting rootfs directory")
var imageFullName *string = flag.String("i", "", "name of the image")

func init() {
	flag.Usage = func() {
		fmt.Printf("Usage: dlrootfs -i <image_name>:[<image_tag>] [-d <rootfs_destination>]\n\n")
		fmt.Printf("Examples:\n")
		fmt.Printf("\tdlrootfs -i ubuntu  #if no tag, use latest\n")
		fmt.Printf("\tdlrootfs -i ubuntu:precise\n")
		fmt.Printf("\tdlrootfs -i dockefile/elasticsearch:latest\n")
		fmt.Printf("Default:\n")
		flag.PrintDefaults()
	}
}

func assertErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {

	flag.Parse()

	if *imageFullName == "" {
		flag.Usage()
		return
	}

	var imageName string
	var imageTag string

	if strings.Contains(*imageFullName, ":") {
		imageName = strings.Split(*imageFullName, ":")[0]
		imageTag = strings.Split(*imageFullName, ":")[1]
	} else {
		imageName = *imageFullName
		imageTag = "latest"
	}

	fmt.Printf("Requesting image: %v:%v\n", imageName, imageTag)

	//resolving endpoint
	registryEndpoint, err := resolveEndpointForImage(imageName)
	assertErr(err)

	fmt.Printf("Endpoint: %v\nAPI: %v\n", registryEndpoint.URL, registryEndpoint.Version)

	session, err := openSession(registryEndpoint)
	assertErr(err)

	//Get back token and endpoint for the repository
	repoData, err := session.GetRepositoryData(imageName)
	assertErr(err)

	fmt.Printf("Fetching: %v (tokens: %v)\n", repoData.Endpoints, repoData.Tokens)

	tagsList, err := session.GetRemoteTags(repoData.Endpoints, imageName, repoData.Tokens)
	assertErr(err)
	imageId := tagsList[imageTag]
	fmt.Printf("Image ID: %v\n", imageId)

	//Download image history (get back all the layers)
	history, err := session.GetRemoteHistory(imageId, repoData.Endpoints[0], repoData.Tokens)
	assertErr(err)

	os.MkdirAll(*rootfsDest, 0777)

	var lastimageData []byte

	cpt := 1
	for i := len(history) - 1; i >= 0; i-- {
		layerId := history[i]

		fmt.Printf("\tDownloading dependant layer %d/%d %v ...\n", cpt, len(history), layerId)
		layerData, imageData, err := downloadImageLayer(session, layerId, repoData.Endpoints[0], repoData.Tokens)
		defer layerData.Close()
		assertErr(err)

		fmt.Printf("\tUntaring layer %v\n", layerId)
		err = archive.Untar(layerData, *rootfsDest, nil)
		assertErr(err)

		fmt.Printf("\tdone %v\n", layerId)
		cpt++

		if i == 0 {
			lastimageData = imageData
		}

	}

	var imageInfo map[string]interface{}
	err = json.Unmarshal(lastimageData, &imageInfo)
	assertErr(err)
	prettyInfo, err := json.MarshalIndent(imageInfo, "", "  ")
	assertErr(err)

	fmt.Printf("All good, %v:%v in %v\n", imageName, imageTag, *rootfsDest)
	fmt.Printf("Image informations: \n %v\n", string(prettyInfo))
}

func openSession(endpoint *registry.Endpoint) (*registry.Session, error) {
	//opening a session
	//empty auth config (probably used only for private repository or private images I guess)
	authConfig := &registry.AuthConfig{}
	var metaHeaders map[string][]string

	return registry.NewSession(authConfig, registry.HTTPRequestFactory(metaHeaders), endpoint, true)
}

func resolveEndpointForImage(imageName string) (*registry.Endpoint, error) {
	hostname, _, err := registry.ResolveRepositoryName(imageName)
	if err != nil {
		return nil, err
	}
	return registry.NewEndpoint(hostname)
}

func downloadImageLayer(session *registry.Session, imageId, endpoint string, tokens []string) (io.ReadCloser, []byte, error) {
	imageData, imageSize, err := session.GetRemoteImageJSON(imageId, endpoint, tokens)
	if err != nil {
		return nil, nil, err
	}
	layerData, err := session.GetRemoteImageLayer(imageId, endpoint, tokens, int64(imageSize))
	return layerData, imageData, err
}
