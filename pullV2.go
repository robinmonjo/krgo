package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

//krgo pull image -r rootfs -v2
//download a flattened docker image from the V2 registry
func (s *registrySession) pullImageV2(imageName, imageTag, rootfsDest string) error {
	return s.downloadImageV2(imageName, imageTag, rootfsDest, false)
}

//krgo pull image -r rootfs -g -v2
//download a docker image from the V1 registry putting each layer in a git branch "on top of each other"
func (s *registrySession) pullRepositoryV2(imageName, imageTag, rootfsDest string) error {
	return s.downloadImageV2(imageName, imageTag, rootfsDest, true)
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
		fmt.Printf("\t%s (%.2f MB) ... ", checksum, float64(job.LayerSize)/ONE_MB)
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
