// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/docker/pkg/tarsum"
	"github.com/docker/docker/registry"
)

/*
  Unused code, prepare the ground for full registry v2 when it's production ready
*/

func generateManifest(gitRepo *gitRepo, imageName, imageTag string) (*registry.ManifestData, error) {
	branches, err := gitRepo.branch()
	if err != nil {
		return nil, err
	}
	var imageChecksums []string = make([]string, len(branches))
	for _, br := range branches {
		checksum := br.imageID()
		sumTypeBytes, err := gitRepo.branchDescription(br)
		if err != nil {
			return nil, err
		}
		imageChecksums[br.number()] = string(sumTypeBytes) + ":" + checksum
	}

	manifest := &registry.ManifestData{
		Name:          imageName,
		Architecture:  "amd64", //unclean but so far looks ok ...
		Tag:           imageTag,
		SchemaVersion: 1,
		FSLayers:      make([]*registry.FSLayer, 0, 4),
	}

	for i, checksum := range imageChecksums {
		if tarsum.VersionLabelForChecksum(checksum) != tarsum.Version1.String() {
			//need to calculate the tarsum V1 for each layer ...
			layerData, err := gitRepo.exportChangeSet(branches[i])
			if err == ErrNoChange {
				continue
			}
			if err != nil {
				return nil, err
			}
			defer layerData.Close()

			tarSum, err := tarsum.NewTarSum(layerData, true, tarsum.Version1)
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(ioutil.Discard, tarSum); err != nil {
				return nil, err
			}

			checksum = tarSum.Sum(nil)
		}
		manifest.FSLayers = append(manifest.FSLayers, &registry.FSLayer{BlobSum: checksum})
	}
	return manifest, nil
}

func (s *registrySession) pushRepositoryV2(imageName, imageTag, rootfs string) error {
	if !isGitRepo(rootfs) {
		return fmt.Errorf("%v not a git repository", rootfs)
	}
	gitRepo, _ := newGitRepo(rootfs)

	endpoint, err := s.V2RegistryEndpoint(s.indexInfo)
	if err != nil {
		return err
	}
	auth, err := s.GetV2Authorization(endpoint, imageName, true)
	if err != nil {
		return err
	}
	fmt.Printf("Registry endpoint: %v\n", endpoint)

	manifest, err := generateManifest(gitRepo, imageName, imageTag)
	if err != nil {
		return err
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "   ")
	if err != nil {
		return err
	}

	branches, err := gitRepo.branch()
	if err != nil {
		return err
	}
	orderedBranches := make([]branch, len(branches))
	for _, br := range branches {
		orderedBranches[br.number()] = br
	}

	for i := len(manifest.FSLayers) - 1; i >= 0; i-- {
		sumStr := manifest.FSLayers[i].BlobSum
		sumParts := strings.SplitN(sumStr, ":", 2)
		if len(sumParts) < 2 {
			return fmt.Errorf("Invalid checksum: %s", sumStr)
		}
		manifestSum := sumParts[1]

		// Call mount blob
		exists, err := s.HeadV2ImageBlob(endpoint, imageName, sumParts[0], manifestSum, auth)
		if err != nil {
			return err
		}

		if !exists {
			fmt.Println("Image doesn't exist")
			layerData, err := gitRepo.exportChangeSet(orderedBranches[i])
			if err != nil {
				return err
			}
			defer layerData.Close()
			fmt.Println(endpoint)
			err = s.PutV2ImageBlob(endpoint, imageName, sumParts[0], sumParts[1], layerData, auth)
			if err != nil {
				return err
			}

			//todo push manifest
		} else {
			fmt.Println("Image already exists")
		}

		// push the manifest
		if err := s.PutV2ImageManifest(endpoint, imageName, imageTag, bytes.NewReader([]byte(manifestBytes)), auth); err != nil {
			return err
		}
	}

	return nil
}
