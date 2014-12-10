package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

const (
	VERSION            string = "1.3.2"
	MAX_DL_CONCURRENCY int    = 7
)

var (
	rootfsDest    *string = flag.String("d", "./rootfs", "destination of the resulting rootfs directory")
	imageFullName *string = flag.String("i", "", "name of the image <repository>/<image>:<tag>")
	credentials   *string = flag.String("u", "", "docker hub credentials: <username>:<password>")
	gitLayering   *bool   = flag.Bool("g", false, "use git layering")
	version       *bool   = flag.Bool("v", false, "display dlrootfs version")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dlrootfs -i <image_name>:[<image_tag>] [-d <rootfs_destination>] [-u <username>:<password>]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs -i ubuntu  #if no tag, use latest\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs -i ubuntu:precise -d ubuntu_rootfs\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs -i dockefile/elasticsearch:latest\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs -i my_repo/my_image:latest -u username:password\n")
		fmt.Fprintf(os.Stderr, "Default:\n")
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

	if *version {
		fmt.Println(VERSION)
		return
	}

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

	session, err := openSession(registryEndpoint, *credentials)
	assertErr(err)

	//Get back token and endpoint for the repository
	repoData, err := session.GetRepositoryData(imageName)
	assertErr(err)

	tagsList, err := session.GetRemoteTags(repoData.Endpoints, imageName, repoData.Tokens)
	assertErr(err)
	imageId := tagsList[imageTag]
	fmt.Printf("Image ID: %v\n", imageId)

	//Download image history (get back all the layers)
	history, err := session.GetRemoteHistory(imageId, repoData.Endpoints[0], repoData.Tokens)
	assertErr(err)

	err = os.MkdirAll(*rootfsDest, 0700)
	assertErr(err)

	var gitRepo *GitRepo
	if *gitLayering {
		gitRepo, err = NewGitRepo(*rootfsDest)
		assertErr(err)
	}

	queue := NewQueue(MAX_DL_CONCURRENCY)

	fmt.Printf("Pulling %d layers:\n", len(history))

	for i := len(history) - 1; i >= 0; i-- {
		layerId := history[i]
		job := NewPullingJob(session, repoData, layerId)
		queue.Enqueue(job)
	}
	<-queue.DoneChan

	fmt.Printf("Downloading layers:\n")

	//no lchown if not on linux
	tarOptions := &archive.TarOptions{NoLchown: false}
	if runtime.GOOS != "linux" {
		tarOptions.NoLchown = true
	}

	cpt := 0

	for i := len(history) - 1; i >= 0; i-- {

		//for each layers
		layerId := history[i]
		fmt.Printf("\t%v ... ", truncateID(layerId))

		if *gitLayering {
			//create a git branch
			_, err = gitRepo.checkoutB("layer" + strconv.Itoa(cpt) + "_" + truncateID(layerId))
			assertErr(err)
		}

		//download and untar the layer
		job := queue.CompletedJobWithID(layerId).(*PullingJob)
		err = archive.Untar(job.LayerData, *rootfsDest, tarOptions)
		job.LayerData.Close()
		assertErr(err)

		//write image info
		var imageInfo map[string]interface{}
		err = json.Unmarshal(job.LayerInfo, &imageInfo)
		assertErr(err)
		prettyInfo, _ := json.MarshalIndent(imageInfo, "", "  ")
		ioutil.WriteFile(*rootfsDest+"/layer_info.json", prettyInfo, 0644)

		if *gitLayering {
			_, err = gitRepo.add(".")
			assertErr(err)
			_, err = gitRepo.commit("adding layer " + strconv.Itoa(cpt))
			assertErr(err)
		}

		cpt++
		fmt.Printf("done\n")
	}

	fmt.Printf("\nRootfs of %v:%v in %v\n", imageName, imageTag, *rootfsDest)
	if *credentials != "" {
		fmt.Printf("WARNING: don't forget to remove your docker hub credentials from your history !!\n")
	}
}

func openSession(endpoint *registry.Endpoint, credentials string) (*registry.Session, error) {
	authConfig := &registry.AuthConfig{}
	if credentials != "" {
		credentialsSplit := strings.SplitN(credentials, ":", 2)
		if len(credentialsSplit) != 2 {
			return nil, fmt.Errorf("Invalid credentials %v", credentials)
		}
		authConfig.Username = credentialsSplit[0]
		authConfig.Password = credentialsSplit[1]
	}

	var metaHeaders map[string][]string

	return registry.NewSession(authConfig, registry.HTTPRequestFactory(metaHeaders), endpoint, true)
}

func resolveEndpointForImage(imageName string) (*registry.Endpoint, error) {
	hostname, _, err := registry.ResolveRepositoryName(imageName)
	if err != nil {
		return nil, err
	}
	return registry.NewEndpoint(hostname, []string{})
}
