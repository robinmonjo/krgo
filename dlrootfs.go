package dlrootfs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

const MAX_DL_CONCURRENCY int = 7

type PullContext struct {
	ImageName       string
	ImageTag        string
	UserCredentials string

	Endpoint *registry.Endpoint
	Session  *registry.Session
	RepoData *registry.RepositoryData

	ImageId      string
	ImageHistory []string
}

func RequestPullContext(imageNameTag, credentials string) (*PullContext, error) {
	context := &PullContext{}

	if strings.Contains(imageNameTag, ":") {
		context.ImageName = strings.Split(imageNameTag, ":")[0]
		context.ImageTag = strings.Split(imageNameTag, ":")[1]
	} else {
		context.ImageName = imageNameTag
		context.ImageTag = "latest"
	}

	hostname, _, err := registry.ResolveRepositoryName(context.ImageName)
	if err != nil {
		return nil, fmt.Errorf("unable to find repository %v", err)
	}
	context.Endpoint, err = registry.NewEndpoint(hostname, []string{})
	if err != nil {
		return nil, err
	}

	authConfig := &registry.AuthConfig{}
	if credentials != "" {
		credentialsSplit := strings.SplitN(credentials, ":", 2)
		if len(credentialsSplit) != 2 {
			return nil, fmt.Errorf("invalid credentials %v", credentials)
		}
		authConfig.Username = credentialsSplit[0]
		authConfig.Password = credentialsSplit[1]
	}

	var metaHeaders map[string][]string

	context.Session, err = registry.NewSession(authConfig, registry.HTTPRequestFactory(metaHeaders), context.Endpoint, true)
	if err != nil {
		return nil, fmt.Errorf("unable to create Docker Hub session %v", err)
	}

	context.RepoData, err = context.Session.GetRepositoryData(context.ImageName)
	if err != nil {
		return nil, fmt.Errorf("unable to get repository data %v", err)
	}

	tagsList, err := context.Session.GetRemoteTags(context.RepoData.Endpoints, context.ImageName, context.RepoData.Tokens)
	if err != nil {
		return nil, fmt.Errorf("unable to find tag list %v", err)
	}

	context.ImageId = tagsList[context.ImageTag]

	//Download image history (get back all the layers)
	context.ImageHistory, err = context.Session.GetRemoteHistory(context.ImageId, context.RepoData.Endpoints[0], context.RepoData.Tokens)
	if err != nil {
		return nil, fmt.Errorf("unable to get back image history %v", err)
	}
	return context, nil
}

func DownloadImage(context *PullContext, rootfsDest string, gitLayering, printProgress bool) error {

	err := os.MkdirAll(rootfsDest, 0700)
	if err != nil {
		return fmt.Errorf("failed to create directory %v: %v", rootfsDest, err)
	}

	var gitRepo *GitRepo
	if gitLayering {
		if gitRepo, err = NewGitRepo(rootfsDest); err != nil {
			return fmt.Errorf("failed to create git repository %v", err)
		}
	}

	queue := NewQueue(MAX_DL_CONCURRENCY)

	if printProgress {
		fmt.Printf("Pulling %d layers:\n", len(context.ImageHistory))
	}

	for i := len(context.ImageHistory) - 1; i >= 0; i-- {
		layerId := context.ImageHistory[i]
		job := NewPullingJob(context.Session, context.RepoData, layerId)
		queue.Enqueue(job)
	}
	<-queue.DoneChan

	if printProgress {
		fmt.Printf("Downloading layers:\n")
	}

	//no lchown if not on linux
	tarOptions := &archive.TarOptions{NoLchown: false}
	if runtime.GOOS != "linux" {
		tarOptions.NoLchown = true
	}

	cpt := 0

	for i := len(context.ImageHistory) - 1; i >= 0; i-- {

		//for each layers
		layerId := context.ImageHistory[i]

		if printProgress {
			fmt.Printf("\t%v ... ", truncateID(layerId))
		}

		if gitLayering {
			//create a git branch
			if _, err = gitRepo.CheckoutB("layer" + strconv.Itoa(cpt) + "_" + truncateID(layerId)); err != nil {
				return fmt.Errorf("failed to checkout %v", err)
			}
		}

		//download and untar the layer
		job := queue.CompletedJobWithID(layerId).(*PullingJob)
		err = archive.Untar(job.LayerData, rootfsDest, tarOptions)
		job.LayerData.Close()
		if err != nil {
			return err
		}

		//write image info
		var layerInfo map[string]interface{}
		err = json.Unmarshal(job.LayerInfo, &layerInfo)
		if err != nil {
			return err
		}

		prettyInfo, _ := json.MarshalIndent(layerInfo, "", "  ")
		ioutil.WriteFile(rootfsDest+"/layer_info.json", prettyInfo, 0644)

		if gitLayering {
			_, err = gitRepo.Add(".")
			if err != nil {
				return fmt.Errorf("failed to add changes %v", err)
			}
			_, err = gitRepo.Commit("adding layer " + strconv.Itoa(cpt))
			if err != nil {
				return fmt.Errorf("failed to commit changes %v", err)
			}
		}

		cpt++

		if printProgress {
			fmt.Printf("done\n")
		}
	}
	return nil
}
