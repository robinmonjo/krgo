package dlrootfs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

const MAX_DL_CONCURRENCY int = 7

type HubSession struct {
	registry.Session
	RepoData *registry.RepositoryData
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

	repoData, err := session.GetRepositoryData(imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository data %v", err)
	}

	return &HubSession{*session, repoData}, nil
}

func (s *HubSession) DownloadFlattenedImage(imageName, imageTag, rootfsDest string, gitLayering, printProgress bool) error {

	tagsList, err := s.GetRemoteTags(s.RepoData.Endpoints, imageName, s.RepoData.Tokens)
	if err != nil {
		return fmt.Errorf("failed to retrieve tag list %v", err)
	}

	imageId := tagsList[imageTag]
	if printProgress {
		fmt.Printf("Image ID: %v\n", imageId)
	}

	//Download image history
	var imageHistory []string
	for _, ep := range s.RepoData.Endpoints {
		imageHistory, err = s.GetRemoteHistory(imageId, ep, s.RepoData.Tokens)
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

	if printProgress {
		fmt.Printf("Pulling %d layers:\n", len(imageHistory))
	}

	for i := len(imageHistory) - 1; i >= 0; i-- {
		layerId := imageHistory[i]
		job := NewPullingJob(s, layerId)
		queue.Enqueue(job)
	}
	<-queue.DoneChan

	if printProgress {
		fmt.Printf("Downloading layers:\n")
	}

	//no lchown if not on linux
	tarOptions := &archive.TarOptions{NoLchown: false}
	if runtime.GOOS != "linux" {
		tarOptions.NoLchown = true
	}

	cpt := 0

	for i := len(imageHistory) - 1; i >= 0; i-- {

		//for each layers
		layerId := imageHistory[i]

		if printProgress {
			fmt.Printf("\t%v ... ", utils.TruncateID(layerId))
		}

		if gitLayering {
			//create a git branch
			if _, err = gitRepo.CheckoutB("layer" + strconv.Itoa(cpt) + "_" + layerId); err != nil {
				return fmt.Errorf("failed to checkout %v", err)
			}
		}

		//download and untar the layer
		job := queue.CompletedJobWithID(layerId).(*PullingJob)
		err = archive.Untar(job.LayerData, rootfsDest, tarOptions)
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
			_, err = gitRepo.Add(".")
			if err != nil {
				return fmt.Errorf("failed to add changes %v", err)
			}
			_, err = gitRepo.Commit("adding layer " + strconv.Itoa(cpt))
			if err != nil {
				return fmt.Errorf("failed to commit changes %v", err)
			}
		}

		cpt++

		if printProgress {
			fmt.Printf("done\n")
		}
	}
	return nil
}

// Pushing specific functions
func ExportChanges(br1, br2, rootfs string) (archive.Archive, error) {

	if !IsGitRepo(rootfs) {
		return nil, fmt.Errorf("%v doesn't appear to be a git repository", rootfs)
	}

	gitRepo, err := NewGitRepo(rootfs)

	diff, err := gitRepo.DiffStatusName(br1, br2)
	if err != nil {
		return nil, fmt.Errorf("failed to diff %v and %v: %v, %v", br1, br2, err, string(diff))
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
	return archive.ExportChanges(rootfs, changes)
}

func (s *HubSession) PushImageLayer(layerData archive.Archive, imageName, imageTag, rootfs string) error {

	//Load image data
	image, err := image.LoadImage(rootfs)
	if err != nil {
		return err
	}

	jsonRaw, err := ioutil.ReadFile(path.Join(rootfs, "json"))
	if err != nil {
		return err
	}

	//4: prepare payload
	imgData := &registry.ImgData{ID: image.ID, Tag: ""}

	fmt.Println("Image data = ", imgData)

	// Register all the images in a repository with the registry
	// If an image is not in this list it will not be associated with the repository
	repoData, err := s.PushImageJSONIndex(imageName, []*registry.ImgData{imgData}, false, nil)
	if err != nil {
		return err
	}

	_, err = s.PushImageJSONIndex(imageName, []*registry.ImgData{imgData}, true, repoData.Endpoints)
	if err != nil {
		return err
	}

	// Send the json
	if err := s.PushImageJSONRegistry(imgData, jsonRaw, repoData.Endpoints[0], repoData.Tokens); err != nil {
		return err
	}

	fmt.Printf("JSON registry\n")
	//fmt.Printf("rendered layer for %s of [%d] size", imgData.ID, layerData.Size)
	checksum, checksumPayload, err := s.PushImageLayerRegistry(imgData.ID, layerData, repoData.Endpoints[0], repoData.Tokens, jsonRaw)
	if err != nil {
		return err
	}

	fmt.Printf("Layer pushed\n")

	imgData.Checksum = checksum
	imgData.ChecksumPayload = checksumPayload
	// Send the checksum
	if err := s.PushImageChecksumRegistry(imgData, repoData.Endpoints[0], repoData.Tokens); err != nil {
		return err
	}
	return nil
}

//PushImageJSONRegistry
//PushImageLayerRegistry
//PushImageChecksumRegistry ??
//func (s *TagStore) pushRepository(r *registry.Sessi when multiple images

func WriteArchiveToFile(archive archive.Archive, dest string) error {
	reader := bufio.NewReader(archive)
	tar, err := os.Create(dest)
	defer tar.Close()

	_, err = reader.WriteTo(tar)
	return err
}
