package dlrootfs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/dockerversion"
	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/tarsum"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

func init() {
	dockerversion.VERSION = "1.4.1" //needed otherwise error 500 on push
}

const MAX_DL_CONCURRENCY int = 7

var PrintOutput bool

type HubSession struct {
	registry.Session
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

	return &HubSession{*session}, nil
}

func (s *HubSession) DownloadFlattenedImage(imageName, imageTag, rootfsDest string, gitLayering bool) error {
	repoData, err := s.GetRepositoryData(imageName)
	if err != nil {
		return fmt.Errorf("failed to get repository data %v", err)
	}

	tagsList, err := s.GetRemoteTags(repoData.Endpoints, imageName, repoData.Tokens)
	if err != nil {
		return fmt.Errorf("failed to retrieve tag list %v", err)
	}

	imageId := tagsList[imageTag]
	_print("Image ID: %v\n", imageId)

	//Download image history
	var imageHistory []string
	for _, ep := range repoData.Endpoints {
		imageHistory, err = s.GetRemoteHistory(imageId, ep, repoData.Tokens)
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
	_print("Pulling %d layers:\n", len(imageHistory))

	for i := len(imageHistory) - 1; i >= 0; i-- {
		layerId := imageHistory[i]
		job := NewPullingJob(s, repoData, layerId)
		queue.Enqueue(job)
	}
	<-queue.DoneChan

	_print("Downloading layers:\n")

	cpt := 0

	for i := len(imageHistory) - 1; i >= 0; i-- {

		//for each layers
		layerId := imageHistory[i]

		_print("\t%v ... ", utils.TruncateID(layerId))

		if gitLayering {
			//create a git branch
			if _, err = gitRepo.CheckoutB("layer" + strconv.Itoa(cpt) + "_" + layerId); err != nil {
				return fmt.Errorf("failed to checkout %v", err)
			}
		}

		//download and untar the layer
		job := queue.CompletedJobWithID(layerId).(*PullingJob)
		err = archive.ApplyLayer(rootfsDest, job.LayerData)
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
		ioutil.WriteFile(path.Join(rootfsDest, "json"), prettyInfo, 0644)
		if gitLayering {
			ioutil.WriteFile(path.Join(rootfsDest, "layersize"), []byte(strconv.Itoa(job.LayerSize)), 0644)
		}

		if gitLayering {
			_, err = gitRepo.AddAllAndCommit(".", "adding layer "+strconv.Itoa(cpt))
			if err != nil {
				return fmt.Errorf("failed to add changes %v", err)
			}
		}

		cpt++

		_print("done\n")
	}
	return nil
}

//Expected changes not to be commited. Changes will be exported from the current branch
func ExportChanges(rootfs string) (archive.Archive, error) {
	if !IsGitRepo(rootfs) {
		return nil, fmt.Errorf("%v doesn't appear to be a git repository", rootfs)
	}

	gitRepo, _ := NewGitRepo(rootfs)
	gitRepo.AddAll(".")

	diff, err := gitRepo.DiffCachedNameStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to diff: %v, %v", err, string(diff))
	}

	var changes []archive.Change

	scanner := bufio.NewScanner(bytes.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()
		dType := strings.SplitN(line, "\t", 2)[0]
		path := "/" + strings.SplitN(line, "\t", 2)[1] // important to consider the / for ExportChanges

		change := archive.Change{Path: path}

		switch dType {
		case DIFF_MODIFIED:
			change.Kind = archive.ChangeModify
		case DIFF_ADDED:
			change.Kind = archive.ChangeAdd
		case DIFF_DELETED:
			change.Kind = archive.ChangeDelete
		}

		changes = append(changes, change)

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("no changes to extract")
	}
	return archive.ExportChanges(rootfs, changes)
}

func (s *HubSession) PushImageLayer(layerData archive.Archive, imageName, imageTag, comment, rootfs string) error {
	if !IsGitRepo(rootfs) {
		return fmt.Errorf("%v doesn't appear to be a git repository", rootfs)
	}

	//Load image data
	image, err := image.LoadImage(rootfs) //reading json file in rootfs
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
	image.Comment = comment

	layer, err := archive.NewTempArchive(layerData, "")
	if err != nil {
		return err
	}
	image.Size = layer.Size

	if err := image.SaveSize(rootfs); err != nil {
		return err
	}

	_print("Image ID: %v\nParent: %v\nChecksum: %v\nLayer size: %v\n", image.ID, image.Parent, image.Checksum, image.Size)

	jsonRaw, err := json.Marshal(image)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(rootfs, "json"), jsonRaw, 0600)
	if err != nil {
		return err
	}

	//Prepare payload
	imgData := &registry.ImgData{ID: image.ID, Tag: imageTag}

	repoData, err := s.PushImageJSONIndex(imageName, []*registry.ImgData{imgData}, false, nil)
	if err != nil {
		return err
	}

	// Send the json
	for _, ep := range repoData.Endpoints {
		if err = s.pushLayer(imageName, layer, imgData, jsonRaw, ep, repoData.Tokens); err == nil {
			break
		}
	}

	if err != nil {
		return err
	}

	_, err = s.PushImageJSONIndex(imageName, []*registry.ImgData{imgData}, true, repoData.Endpoints)
	if err != nil {
		return err
	}

	//commit the changes in a new branch
	gitRepo, _ := NewGitRepo(rootfs)
	brNumber, _ := gitRepo.CountBranches()
	br := "layer" + strconv.Itoa(brNumber-1) + "_" + image.ID
	if _, err = gitRepo.CheckoutB(br); err != nil {
		return fmt.Errorf("failed to checkout %v", err)
	}
	if _, err = gitRepo.AddAllAndCommit(".", comment); err != nil {
		return fmt.Errorf("failed to locally commit pushed changes %v", err)
	}

	_print("Uploaded layer is in the %v branch\n", br)

	return nil
}

func (s *HubSession) pushLayer(imageName string, layer archive.Archive, imgData *registry.ImgData, jsonRaw []byte, ep string, tokens []string) error {
	if err := s.PushImageJSONRegistry(imgData, jsonRaw, ep, tokens); err != nil {
		return err
	}

	checksum, checksumPayload, err := s.PushImageLayerRegistry(imgData.ID, layer, ep, tokens, jsonRaw)
	if err != nil {
		return err
	}

	imgData.Checksum = checksum
	imgData.ChecksumPayload = checksumPayload

	// Send the checksum
	if err := s.PushImageChecksumRegistry(imgData, ep, tokens); err != nil {
		return err
	}

	if err := s.PushRegistryTag(imageName, imgData.ID, imgData.Tag, ep, tokens); err != nil {
		return err
	}
	return nil
}
