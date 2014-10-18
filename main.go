package main

import (
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/docker/docker/registry"
)

const (
	IMAGE_ID    string = "9942dd43ff211ba917d03637006a83934e847c003bef900e4808be8021dca7bd"
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
	assertErr(err)

	tokens := data.Tokens
	repoEndpoint := data.Endpoints[0]

	//Download image history (get back all the layers)
	history, err := session.GetRemoteHistory(IMAGE_ID, repoEndpoint, tokens)
	assertErr(err)

	log.Println("Image", IMAGE_ID, "is made of", len(history), "layers:", history)

	os.MkdirAll(ROOTFS_DEST, 0700)

	tarLayers := make([]string, 0, len(history))
	var wg sync.WaitGroup

	for i := len(history) - 1; i >= 0; i-- {
		imageId := history[i]

		fileName := ROOTFS_DEST + "/layer_" + imageId + ".tar"
		tarLayers = append(tarLayers, fileName)
		wg.Add(1)

		go func() {
			defer wg.Done()

			log.Println("Downloading layer", imageId, "...")
			layerData, err := downloadImageLayer(session, imageId, repoEndpoint, tokens)
			defer layerData.Close()
			assertErr(err)

			out, err := os.Create(fileName)
			defer out.Close()

			io.Copy(out, layerData)

			log.Println("done", imageId)
		}()
	}
	wg.Wait()

	log.Print("Extracting layers in a single rootfs ... ")

	for _, tarLayer := range tarLayers {
		err := untarFile(tarLayer)
		assertErr(err)
		os.Remove(tarLayer)
	}

	log.Println("All good")

}

func untarFile(tarfile string) error {
	out, err := exec.Command("tar", "xf", tarfile, "-C", ROOTFS_DEST).CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}
	return nil
}

func downloadImageLayer(session *registry.Session, imageId, endpoint string, tokens []string) (io.ReadCloser, error) {
	//Get back image information
	_, imgSize, err := session.GetRemoteImageJSON(imageId, endpoint, tokens)
	if err != nil {
		return nil, err
	}
	return session.GetRemoteImageLayer(imageId, endpoint, tokens, int64(imgSize))
}
