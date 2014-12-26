package dlrootfs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

const MAX_DL_CONCURRENCY int = 7

type HubSession struct {
	registry.Session
	RepoData *registry.RepositoryData
}

func NewHubSession(imageName, userName, password string) (*HubSession, error) {
	hostname, _, err := registry.ResolveRepositoryName(imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository for image %v: %v", imageName, err)
	}
	endpoint, err := registry.NewEndpoint(hostname, []string{})
	if err != nil {
		return nil, err
	}

	authConfig := &registry.AuthConfig{}
	if userName != "" && password != "" {
		authConfig.Username = userName
		authConfig.Password = password
	}

	var metaHeaders map[string][]string

	session, err := registry.NewSession(authConfig, registry.HTTPRequestFactory(metaHeaders), endpoint, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker hub session %v", err)
	}

	repoData, err := session.GetRepositoryData(imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository data %v", err)
	}

	return &HubSession{*session, repoData}, nil
}

func (s *HubSession) DownloadFlattenedImage(imageName, imageTag, rootfsDest string, gitLayering, printProgress bool) error {

	tagsList, err := s.GetRemoteTags(s.RepoData.Endpoints, imageName, s.RepoData.Tokens)
	if err != nil {
		return fmt.Errorf("failed to retrieve tag list %v", err)
	}

	imageId := tagsList[imageTag]
	if printProgress {
		fmt.Printf("Image ID: %v\n", imageId)
	}

	//Download image history
	var imageHistory []string
	for _, ep := range s.RepoData.Endpoints {
		imageHistory, err = s.GetRemoteHistory(imageId, ep, s.RepoData.Tokens)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("failed to get back image history %v", err)
	}

	err = os.MkdirAll(rootfsDest, 0700)
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
		fmt.Printf("Pulling %d layers:\n", len(imageHistory))
	}

	for i := len(imageHistory) - 1; i >= 0; i-- {
		layerId := imageHistory[i]
		job := NewPullingJob(s, layerId)
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

	for i := len(imageHistory) - 1; i >= 0; i-- {

		//for each layers
		layerId := imageHistory[i]

		if printProgress {
			fmt.Printf("\t%v ... ", truncateID(layerId))
		}

		if gitLayering {
			//create a git branch
			if _, err = gitRepo.CheckoutB("layer" + strconv.Itoa(cpt) + "_" + layerId); err != nil {
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
		ioutil.WriteFile(path.Join(rootfsDest, "image.json"), prettyInfo, 0644)
		if gitLayering {
			ioutil.WriteFile(path.Join(rootfsDest, "layersize"), []byte(strconv.Itoa(job.LayerSize)), 0644)
		}

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
