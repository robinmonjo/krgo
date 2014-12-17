package dlrootfs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/pkg/archive"
)

func ExportChanges(br1, br2, rootfs string) (archive.Archive, error) {

	if !IsGitRepo(rootfs) {
		return nil, fmt.Errorf("%v doesn't appear to be a git repository", rootfs)
	}

	gitRepo, err := NewGitRepo(rootfs)

	diff, err := gitRepo.DiffStatusName(br1, br2)
	if err != nil {
		return nil, fmt.Errorf("failed to diff %v and %v: %v", br1, br2, err)
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

/*func PushImageLayer() {
  //1: define a new id from the commit hash GenerateRandomID in docker utils
  //2: load layer_info.json and change the id
  //3: get back raw json
  jsonRaw, err := ioutil.ReadFile(path.Join(s.graph.Root, imgID, "json")) //layerinfo.json

  //4: prepare payload
  imgData := &registry.ImgData{
    ID: imgID,
  }

  // Send the json
  if err := r.PushImageJSONRegistry(imgData, jsonRaw, ep, token); err != nil {
    if err == registry.ErrAlreadyExists {
      //image already pushed, skipping
      return "", nil
    }
    return "", err
  }

  log.Debugf("rendered layer for %s of [%d] size", imgData.ID, layerData.Size)

  checksum, checksumPayload, err := r.PushImageLayerRegistry(imgData.ID, utils.ProgressReader(layerData, int(layerData.Size), out, sf, false, utils.TruncateID(imgData.ID), "Pushing"), ep, token, jsonRaw)
  if err != nil {
    return "", err
  }
  imgData.Checksum = checksum
  imgData.ChecksumPayload = checksumPayload
  // Send the checksum
  if err := r.PushImageChecksumRegistry(imgData, ep, token); err != nil {
    return "", err
  }

}

//PushImageJSONRegistry
//PushImageLayerRegistry
//PushImageChecksumRegistry ??
//func (s *TagStore) pushRepository(r *registry.Sessi when multiple images


*/

func WriteArchiveToFile(archive archive.Archive, dest string) error {
	reader := bufio.NewReader(archive)
	tar, err := os.Create(dest)
	defer tar.Close()

	_, err = reader.WriteTo(tar)
	return err
}
