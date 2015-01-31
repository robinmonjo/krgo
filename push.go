package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/docker/docker/registry"
)

func (s *registrySession) pushRepository(imageName, imageTag, rootfs string) error {
	if !isGitRepo(rootfs) {
		return fmt.Errorf("%v not a git repository", rootfs)
	}
	gitRepo, _ := newGitRepo(rootfs)

	branches, err := gitRepo.branch()
	if err != nil {
		return err
	}
	var imageIds []string = make([]string, len(branches), len(branches))
	for _, br := range branches {
		idx, _ := exportLayerNumberFromBranch(br)
		imageIds[idx] = strings.Split(br, "_")[2] //branch format layer_N_imageId
	}

	fmt.Printf("Pushing %d layers:\n", len(imageIds))

	//Push image index
	var imageIndex []*registry.ImgData
	for _, id := range imageIds {
		imageIndex = append(imageIndex, &registry.ImgData{ID: id, Tag: imageTag})
	}
	repoData, err := s.PushImageJSONIndex(imageName, imageIndex, false, nil)
	if err != nil {
		return err
	}

	ep := repoData.Endpoints[0]
	//make sure existing branches are pushed
	for i, imageId := range imageIds {
		fmt.Printf("\t%v ... ", imageId)
		if s.LookupRemoteImage(imageId, ep, repoData.Tokens) {
			fmt.Printf("done (already pushed)\n")
		} else {
			err = s.pushImageLayer(gitRepo, branches[i], imageId, ep, repoData.Tokens)
			if err != nil {
				if err == registry.ErrAlreadyExists {
					fmt.Printf("done (already pushed)\n")
				} else {
					return err
				}
			} else {
				fmt.Printf("done\n")
			}
		}

		//push tag
		if err := s.PushRegistryTag(imageName, imageId, imageTag, ep, repoData.Tokens); err != nil {
			return err
		}
	}

	//Finalize push
	if _, err = s.PushImageJSONIndex(imageName, imageIndex, true, repoData.Endpoints); err != nil {
		return err
	}
	return nil
}

func (s *registrySession) pushImageLayer(gitRepo *gitRepo, branch, imgID, ep string, token []string) error {
	if _, err := gitRepo.checkout(branch); err != nil {
		return err
	}

	jsonRaw, err := ioutil.ReadFile(path.Join(gitRepo.Path, "json"))
	if err != nil {
		return err
	}

	imgData := &registry.ImgData{
		ID: imgID,
	}

	// Send the json
	if err := s.PushImageJSONRegistry(imgData, jsonRaw, ep, token); err != nil {
		return err
	}

	layerData, err := gitRepo.exportChangeSet(branch)
	if err != nil {
		return err
	}
	defer layerData.Close()

	checksum, checksumPayload, err := s.PushImageLayerRegistry(imgID, layerData, ep, token, jsonRaw)
	if err != nil {
		return err
	}
	imgData.Checksum = checksum
	imgData.ChecksumPayload = checksumPayload

	return s.PushImageChecksumRegistry(imgData, ep, token)
}
