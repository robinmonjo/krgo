package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"

	"github.com/docker/docker/pkg/archive"
)

const MAX_DL_CONCURRENCY int = 7

//download a flattened dowker image
func (s *hubSession) pullImage(imageName, imageTag, rootfsDest string) error {
	return s.downloadImage(imageName, imageTag, rootfsDest, false)
}

//download an image putting each layer in a git branch "on top of each other"
func (s *hubSession) pullRepository(imageName, imageTag, rootfsDest string) error {
	return s.downloadImage(imageName, imageTag, rootfsDest, true)
}

func (s *hubSession) downloadImage(imageName, imageTag, rootfsDest string, gitLayering bool) error {
	repoData, err := s.GetRepositoryData(imageName)
	if err != nil {
		return err
	}

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
		layerId := imageHistory[i]

		fmt.Printf("\t%v ... ", layerId)

		if gitLayering {
			//create a git branch
			if _, err = gitRepo.checkoutB("layer_" + strconv.Itoa(cpt) + "_" + layerId); err != nil {
				return err
			}
		}

		//download and untar the layer
		job := queue.CompletedJobWithID(layerId).(*PullingJob)
		err = archive.ApplyLayer(rootfsDest, job.LayerData)
		job.LayerData.Close()
		if err != nil {
			return err
		}

		ioutil.WriteFile(path.Join(rootfsDest, "json"), job.LayerInfo, 0644)
		if gitLayering {
			ioutil.WriteFile(path.Join(rootfsDest, "layersize"), []byte(strconv.Itoa(job.LayerSize)), 0644)
		}

		if gitLayering {
			_, err = gitRepo.addAllAndCommit("adding layer " + strconv.Itoa(cpt))
			if err != nil {
				return err
			}
		}

		cpt++

		fmt.Printf("done\n")
	}
	return nil
}
