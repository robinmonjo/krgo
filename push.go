package dlrootfs

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/docker/docker/pkg/archive"
)

func ExportChanges(br1, br2, rootfs string) (archive.Archive, error) {
	//if not git repo, return

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
		path := strings.SplitN(line, "\t", 2)[1]

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
