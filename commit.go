package dlrootfs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/tarsum"
	"github.com/docker/docker/utils"
)

func CommitChanges(rootfs, message string) error {
	if !IsGitRepo(rootfs) {
		return fmt.Errorf("%v doesn't appear to be a git repository", rootfs)
	}
	gitRepo, _ := NewGitRepo(rootfs)

	layerData, err := gitRepo.ExportUncommitedChangeSet()
	if err != nil {
		return err
	}

	//Load image data
	image, err := image.LoadImage(gitRepo.Path) //reading json file in rootfs
	if err != nil {
		return err
	}

	layerTarSum, err := tarsum.NewTarSum(layerData, true, tarsum.VersionDev)
	if err != nil {
		return err
	}

	//fill new infos
	image.Checksum = layerTarSum.Sum(nil)
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
	brNumber, _ := gitRepo.CountBranches()
	br := "layer" + strconv.Itoa(brNumber) + "_" + image.ID
	if _, err = gitRepo.CheckoutB(br); err != nil {
		return fmt.Errorf("failed to checkout %v", err)
	}
	if b, err := gitRepo.AddAllAndCommit(message); err != nil {
		_print(string(b) + "\n")
		return fmt.Errorf("failed to locally commit pushed changes %v", err)
	}

	_print("Your changes and some additional metadata have been commited on branch %v\nSummary:\n", br)
	_print("\tImage ID: %v\n\tParent: %v\n\tChecksum: %v\n\tLayer size: %v\n", image.ID, image.Parent, image.Checksum, image.Size)

	return nil
}
