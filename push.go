package dlrootfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

func (s *HubSession) PushRepository(imageName, imageTag, rootfs string) error {
	if !IsGitRepo(rootfs) {
		return fmt.Errorf("%v doesn't appear to be a git repository", rootfs)
	}
	gitRepo, _ := NewGitRepo(rootfs)

	var imageIds []string
	branches, err := gitRepo.Branch()
	if err != nil {
		return err
	}
	for _, br := range branches {
		imageIds = append(imageIds, strings.Split(br, "_")[1]) //branch format layerN_imageId
	}

	_print("Repository has %d layers\n", len(imageIds))

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
		_print("%v ... ", imageId)
		if s.LookupRemoteImage(imageId, ep, repoData.Tokens) {
			_print("already pushed\n")
		} else {
			err = s.pushImageLayer(gitRepo, branches[i], imageId, ep, repoData.Tokens)
			if err != nil {
				return err
			}
			_print("pushed :)\n")
		}

		//push tag
		if err := s.PushRegistryTag(imageName, imageId, imageTag, ep, repoData.Tokens); err != nil {
			return err
		}
	}

	//push the new uncommited changes

	//Finalize push
	_, err = s.PushImageJSONIndex(imageName, imageIndex, true, repoData.Endpoints)
	if err != nil {
		return err
	}

	return nil
}

func (s *HubSession) pushImageLayer(gitRepo *GitRepo, branch, imgID, ep string, token []string) error {
	if _, err := gitRepo.Checkout(branch); err != nil {
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
		if err == registry.ErrAlreadyExists {
			_print("Image already pushed")
			return nil
		}
		return err
	}

	layerData, err := gitRepo.ExportChangeSet(branch)
	if err != nil {
		return err
	}
	layer, err := archive.NewTempArchive(layerData, "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(layer.Name())

	checksum, checksumPayload, err := s.PushImageLayerRegistry(imgID, layer, ep, token, jsonRaw)
	if err != nil {
		return err
	}
	imgData.Checksum = checksum
	imgData.ChecksumPayload = checksumPayload
	// Send the checksum
	return s.PushImageChecksumRegistry(imgData, ep, token)
}
