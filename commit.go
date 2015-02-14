package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
	//"github.com/docker/docker/pkg/tarsum"
	"github.com/docker/docker/utils"
)

//krgo commit -r rootfs
//commit current changes in a new properly formated branch ready for pushing
func commitChanges(rootfs, message string) error {
	if !isGitRepo(rootfs) {
		return fmt.Errorf("%v not a git repository", rootfs)
	}
	gitRepo, _ := newGitRepo(rootfs)

	layerData, err := gitRepo.exportUncommitedChangeSet()
	if err != nil {
		return err
	}
	defer layerData.Close()

	//Load image data
	image, err := image.LoadImage(gitRepo.Path) //reading json file in rootfs
	if err != nil {
		return err
	}

	//fill new infos
	//image.Checksum = layerTarSum.Sum(nil)
	image.Parent = image.ID
	image.ID = utils.GenerateRandomID()
	image.Created = time.Now()
	image.Comment = message

	layer, err := archive.NewTempArchive(layerData, "")
	if err != nil {
		return err
	}
	image.Size = layer.Size
	os.RemoveAll(layer.Name())

	if err := image.SaveSize(rootfs); err != nil {
		return err
	}

	jsonRaw, err := json.Marshal(image)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(rootfs, "json"), jsonRaw, 0600)
	if err != nil {
		return err
	}

	//commit the changes in a new branch
	n, _ := gitRepo.countBranch()
	br := newBranch(n, image.ID)
	if _, err = gitRepo.checkoutB(br); err != nil {
		return err
	}
	if _, err := gitRepo.addAllAndCommit(message); err != nil {
		return err
	}

	fmt.Printf("Changes commited in %v\n", br)
	fmt.Printf("Image ID: %v\nParent: %v\nChecksum: %v\nLayer size: %v\n", image.ID, image.Parent /*image.Checksum*/, "lol", image.Size)

	return nil
}
