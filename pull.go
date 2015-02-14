package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

const MAX_DL_CONCURRENCY = 7

//cargo pull image -r rootfs
//download a flattened docker image from the V1 registry
func (s *registrySession) pullImage(imageName, imageTag, rootfsDest string) error {
	return s.downloadImage(imageName, imageTag, rootfsDest, false, false)
}

//cargo pull image -r rootfs -v2
//download a flattened docker image from the V2 registry
func (s *registrySession) pullImageV2(imageName, imageTag, rootfsDest string) error {
	return s.downloadImage(imageName, imageTag, rootfsDest, false, true)
}

//cargo pull image -r rootfs -g
//download a docker image from the V1 registry putting each layer in a git branch "on top of each other"
func (s *registrySession) pullRepository(imageName, imageTag, rootfsDest string) error {
	return s.downloadImage(imageName, imageTag, rootfsDest, true, false)
}

//cargo pull image -r rootfs -g -v2
//download a docker image from the V1 registry putting each layer in a git branch "on top of each other"
func (s *registrySession) pullRepositoryV2(imageName, imageTag, rootfsDest string) error {
	return s.downloadImage(imageName, imageTag, rootfsDest, true, true)
}

func (s *registrySession) downloadImage(imageName, imageTag, rootfsDest string, gitLayering, useV2Reg bool) error {
	if useV2Reg {
		return s.downloadImageV2(imageName, imageTag, rootfsDest, gitLayering)
	}
	return s.downloadImageV1(imageName, imageTag, rootfsDest, gitLayering)
}

//pulling using V1 registry
func (s *registrySession) downloadImageV1(imageName, imageTag, rootfsDest string, gitLayering bool) error {
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
		fmt.Printf("\t%s (%.2f MB) ... ", layerID, float64(job.LayerSize)/1000000.0)
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

//pulling using V2 registry (much nicer !)
func (s *registrySession) downloadImageV2(imageName, imageTag, rootfsDest string, gitLayering bool) error {
	endpoint, err := s.V2RegistryEndpoint(s.indexInfo)
	if err != nil {
		return err
	}
	auth, err := s.GetV2Authorization(endpoint, imageName, true)
	if err != nil {
		return err
	}
	fmt.Printf("Registry endpoint: %v\n", endpoint)

	rawManifest, err := s.GetV2ImageManifest(endpoint, imageName, imageTag, auth)
	if err != nil {
		return err
	}

	var manifest registry.ManifestData
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
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
	fmt.Printf("Manifest contains %d layers, try to cleanup ...\n", len(manifest.FSLayers))
	cleanupManifest(&manifest)
	fmt.Printf("Pulling %d layers:\n", len(manifest.FSLayers))

	for i := len(manifest.FSLayers) - 1; i >= 0; i-- {
		sumStr := manifest.FSLayers[i].BlobSum
		job := NewPullingV2Job(s, endpoint, auth, imageName, sumStr)
		queue.Enqueue(job)
	}
	<-queue.DoneChan

	fmt.Printf("Downloading layers:\n")
	cpt := 0
	for i := len(manifest.FSLayers) - 1; i >= 0; i-- {
		sumStr := manifest.FSLayers[i].BlobSum
		sumType := strings.Split(sumStr, ":")[0]
		checksum := strings.Split(sumStr, ":")[1]

		if gitLayering {
			//create a git branch
			br := newBranch(cpt, checksum)
			if _, err = gitRepo.checkoutB(br); err != nil {
				return err
			}
			//set tarsum info into branch description
			if err := gitRepo.describeBranch(br, sumType); err != nil {
				return err
			}
		}

		job := queue.CompletedJobWithID(sumStr).(*PullingV2Job)
		fmt.Printf("\t%s (%.2f MB) ... ", checksum, float64(job.LayerSize)/1000000.0)
		_, err = archive.ApplyLayer(rootfsDest, ioutil.NopCloser(job.LayerTarSumReader))
		if err != nil {
			return err
		}
		finalChecksum := job.LayerTarSumReader.Sum(nil)
		job.LayerDataReader.Close()

		if gitLayering {
			if _, err = gitRepo.addAllAndCommit("adding layer " + checksum); err != nil {
				return err
			}
		}

		verified := strings.EqualFold(finalChecksum, sumStr)
		fmt.Printf("done (tarsum verified: %v)\n", verified)

		cpt++
	}
	return nil
}

//Layers are now addressed by content, i.e identified by their tarsum (https://github.com/docker/docker-registry/issues/612)
//v1 registry required to push the layer json, that made a lot of "duplicated layer"
//So images manifests contain duplicated layers (layers with same content and then same tarsum), we can clean them up
func cleanupManifest(manifest *registry.ManifestData) {
	found := make(map[string]bool)
	cleanFSLayers := []*registry.FSLayer{}
	for _, layer := range manifest.FSLayers {
		if !found[layer.BlobSum] {
			found[layer.BlobSum] = true
			cleanFSLayers = append(cleanFSLayers, &registry.FSLayer{BlobSum: layer.BlobSum})
		}
	}
	manifest.FSLayers = cleanFSLayers
}
