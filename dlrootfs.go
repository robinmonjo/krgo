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

	_ "github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
	"github.com/docker/docker/utils"
)

const MAX_DL_CONCURRENCY int = 7

// Global context needed to connect to the docker hub
type HubContext struct {
	ImageName       string
	ImageTag        string
	UserCredentials string

	RepositoryHostname string

	Session  *registry.Session
	RepoData *registry.RepositoryData
}

// Pulling image specific context
type PullContext struct {
	HubContext

	ImageId      string
	ImageHistory []string
}

func initHubContext(imageNameTag, credentials string) (*HubContext, error) {
	context := &HubContext{}

	context.ImageName, context.ImageTag = extractImageNameTag(imageNameTag)

	hostname, _, err := registry.ResolveRepositoryName(context.ImageName)
	context.RepositoryHostname = hostname
	if err != nil {
		return nil, fmt.Errorf("unable to find repository %v", err)
	}
	endpoint, err := registry.NewEndpoint(hostname, []string{})
	if err != nil {
		return nil, err
	}

	authConfig := &registry.AuthConfig{}
	if credentials != "" {
		credentialsSplit := strings.SplitN(credentials, ":", 2)
		if len(credentialsSplit) != 2 {
			return nil, fmt.Errorf("invalid credentials %v", credentials)
		}
		authConfig.Username = credentialsSplit[0]
		authConfig.Password = credentialsSplit[1]
	}

	var metaHeaders map[string][]string

	context.Session, err = registry.NewSession(authConfig, registry.HTTPRequestFactory(metaHeaders), endpoint, true)
	if err != nil {
		return nil, fmt.Errorf("unable to create Docker Hub session %v", err)
	}

	context.RepoData, err = context.Session.GetRepositoryData(context.ImageName)
	if err != nil {
		return nil, fmt.Errorf("unable to get repository data %v", err)
	}
	return context, nil
}

// Pulling specific functions

func InitPullContext(imageNameTag, credentials string) (*PullContext, error) {
	hubContext, err := initHubContext(imageNameTag, credentials)
	if err != nil {
		return nil, err
	}

	context := &PullContext{HubContext: *hubContext}
	tagsList, err := context.Session.GetRemoteTags(context.RepoData.Endpoints, context.ImageName, context.RepoData.Tokens)
	if err != nil {
		return nil, fmt.Errorf("unable to find tag list %v", err)
	}

	context.ImageId = tagsList[context.ImageTag]

	//Download image history (get back all the layers)
	context.ImageHistory, err = context.Session.GetRemoteHistory(context.ImageId, context.RepoData.Endpoints[0], context.RepoData.Tokens)
	if err != nil {
		return nil, fmt.Errorf("unable to get back image history %v", err)
	}
	return context, nil
}

func DownloadImage(context *PullContext, rootfsDest string, gitLayering, printProgress bool) error {

	err := os.MkdirAll(rootfsDest, 0700)
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
		fmt.Printf("Pulling %d layers:\n", len(context.ImageHistory))
	}

	for i := len(context.ImageHistory) - 1; i >= 0; i-- {
		layerId := context.ImageHistory[i]
		job := NewPullingJob(context.Session, context.RepoData, layerId)
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

	for i := len(context.ImageHistory) - 1; i >= 0; i-- {

		//for each layers
		layerId := context.ImageHistory[i]

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
		ioutil.WriteFile(path.Join(rootfsDest, "image.json"), prettyInfo, 0644)

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

func PushImageLayer(imageNameTag, rootfs, credentials string, layerData archive.Archive) error {

	context, err := initHubContext(imageNameTag, credentials)
	if err != nil {
		return err
	}

	//Load image data
	image, err := LoadImageFromJson(path.Join(rootfs, "image.json"))
	if err != nil {
		return err
	}

	_, imageTag := extractImageNameTag(imageNameTag)
	jsonRaw, err := ioutil.ReadFile(path.Join(rootfs, "image.json"))

	//4: prepare payload
	imgData := &registry.ImgData{ID: image.ID, Tag: imageTag}
	if imageTag == "latest" {
		imgData.Tag = ""
	}

	fmt.Println("Image data = ", imgData)

	// Register all the images in a repository with the registry
	// If an image is not in this list it will not be associated with the repository
	repoData, err := context.Session.PushImageJSONIndex("robinmonjo/debian", []*registry.ImgData{imgData}, false, nil)
	if err != nil {
		return err
	}

	_, err = context.Session.PushImageJSONIndex("robinmonjo/debian", []*registry.ImgData{imgData}, true, repoData.Endpoints)
	if err != nil {
		return err
	}

	// Send the json
	if err := context.Session.PushImageJSONRegistry(imgData, jsonRaw, repoData.Endpoints[0], repoData.Tokens); err != nil {
		return err
	}

	fmt.Printf("JSON registry\n")
	//fmt.Printf("rendered layer for %s of [%d] size", imgData.ID, layerData.Size)
	tmpArchive, err := archive.NewTempArchive(layerData, "/tmp/")
	if err != nil {
		return err
	}
	checksum, checksumPayload, err := context.Session.PushImageLayerRegistry(imgData.ID, tmpArchive, repoData.Endpoints[0], repoData.Tokens, jsonRaw)
	if err != nil {
		return err
	}

	fmt.Printf("Layer pushed\n")

	imgData.Checksum = checksum
	imgData.ChecksumPayload = checksumPayload
	// Send the checksum
	if err := context.Session.PushImageChecksumRegistry(imgData, repoData.Endpoints[0], repoData.Tokens); err != nil {
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
