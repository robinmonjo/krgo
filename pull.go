package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/docker/docker/pkg/archive"
)

const (
	MAX_DL_CONCURRENCY = 7
	ONE_MB             = 1000000
)

//krgo pull image -r rootfs
//download a flattened docker image from the V1 registry
func (s *registrySession) pullImage(imageName, imageTag, rootfsDest string) error {
	return s.downloadImage(imageName, imageTag, rootfsDest, false)
}

//krgo pull image -r rootfs -g
//download a docker image from the V1 registry putting each layer in a git branch "on top of each other"
func (s *registrySession) pullRepository(imageName, imageTag, rootfsDest string) error {
	return s.downloadImage(imageName, imageTag, rootfsDest, true)
}

//pulling using V1 registry
func (s *registrySession) downloadImage(imageName, imageTag, rootfsDest string, gitLayering bool) error {
	repoData, err := s.GetRepositoryData(imageName)
	if err != nil {
		return err
	}
	fmt.Printf("Registry endpoint: %v\n", repoData.Endpoints)

	tagsList, err := s.GetRemoteTags(repoData.Endpoints, imageName, repoData.Tokens)
	if err != nil {
		return err
	}

	imageId := tagsList[imageTag]
	fmt.Printf("Image ID: %v\n", imageId)

	//Download image history
	var imageHistory []string
	for _, ep := range repoData.Endpoints {
		imageHistory, err = s.GetRemoteHistory(imageId, ep, repoData.Tokens)
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}

	err = os.MkdirAll(rootfsDest, 0700)
	if err != nil {
		return err
	}

	var gitRepo *gitRepo
	if gitLayering {
		if gitRepo, err = newGitRepo(rootfsDest); err != nil {
			return err
		}
	}

	queue := NewQueue(MAX_DL_CONCURRENCY)
	fmt.Printf("Pulling %d layers:\n", len(imageHistory))

	for i := len(imageHistory) - 1; i >= 0; i-- {
		layerId := imageHistory[i]
		job := NewPullingJob(s, repoData, layerId)
		queue.Enqueue(job)
	}
	<-queue.DoneChan

	fmt.Printf("Downloading layers:\n")

	cpt := 0

	for i := len(imageHistory) - 1; i >= 0; i-- {

		//for each layers
		layerID := imageHistory[i]

		if gitLayering {
			//create a git branch
			if _, err = gitRepo.checkoutB(newBranch(cpt, layerID)); err != nil {
				return err
			}
		}

		//download and untar the layer
		job := queue.CompletedJobWithID(layerID).(*PullingJob)
		fmt.Printf("\t%s (%.2f MB) ... ", layerID, float64(job.LayerSize)/ONE_MB)
		_, err = archive.ApplyLayer(rootfsDest, job.LayerData)
		job.LayerData.Close()
		if err != nil {
			return err
		}

		ioutil.WriteFile(path.Join(rootfsDest, "json"), job.LayerInfo, 0644)
		if gitLayering {
			ioutil.WriteFile(path.Join(rootfsDest, "layersize"), []byte(strconv.Itoa(job.LayerSize)), 0644)
		}

		if gitLayering {
			if _, err = gitRepo.addAllAndCommit("adding layer " + layerID); err != nil {
				return err
			}
		}

		cpt++

		fmt.Printf("done\n")
	}
	return nil
}
