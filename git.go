package dlrootfs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/pkg/archive"
)

const (
	DIFF_ADDED    string = "A"
	DIFF_MODIFIED string = "M"
	DIFF_DELETED  string = "D"
)

type gitRepo struct {
	Path string
}

func isGitRepo(path string) bool {
	_, err := os.Stat(path + "/.git")
	return err == nil
}

func newGitRepo(path string) (*gitRepo, error) {
	r := &gitRepo{Path: path}

	if isGitRepo(r.Path) {
		return r, nil
	}

	_, err := r.exec("init", path)
	if err != nil {
		return nil, err
	}

	email, _ := r.userConfig("email")
	if len(email) == 0 {
		_, err = r.execInWorkTree("config", "user.email", "fake@dlrootfs.com")
		if err != nil {
			return nil, err
		}
	}

	name, _ := r.userConfig("name")
	if len(name) == 0 {
		_, err = r.execInWorkTree("config", "user.name", "dlrootfs")
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *gitRepo) userConfig(key string) ([]byte, error) {
	return r.exec("config", "user."+key)
}

func (r *gitRepo) checkout(branch string) ([]byte, error) {
	return r.execInWorkTree("checkout", branch)
}

func (r *gitRepo) checkoutB(branch string) ([]byte, error) {
	return r.execInWorkTree("checkout", "-b", branch)
}

func (r *gitRepo) addAllAndCommit(file, message string) ([]byte, error) {
	bAdd, err := r.addAll(file)
	if err != nil {
		return bAdd, err
	}
	bCi, err := r.commit(message)
	return append(bAdd, bCi...), err
}

func (r *gitRepo) addAll(file string) ([]byte, error) {
	return r.execInWorkTree("add", file, "--all")
}

func (r *gitRepo) commit(message string) ([]byte, error) {
	return r.execInWorkTree("commit", "-m", message)
}

func (r *gitRepo) branch() ([]string, error) {
	b, err := r.execInWorkTree("branch")
	if err != nil {
		return nil, err
	}
	brs := strings.Split(string(b), "\n")
	return brs[:len(brs)-1], nil //remove the last empty line
}

func (r *gitRepo) currentBranch() (string, error) {
	b, err := r.execInWorkTree("rev-parse", "--abbrev-ref", "HEAD")
	return string(b), err
}

func (r *gitRepo) imageIds() ([]string, error) {
	branches, err := r.branch()
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, br := range branches {
		ids = append(ids, strings.Split(br, "_")[1]) //branch format layerN_imageId
	}
	return ids, nil
}

func (r *gitRepo) stash() ([]byte, error) {
	return r.execInWorkTree("stash")
}

func (r *gitRepo) unstash() ([]byte, error) {
	return r.execInWorkTree("stash", "apply")
}

func (r *gitRepo) countBranches() (int, error) {
	branches, err := r.branch()
	if err != nil {
		return -1, err
	}
	return len(branches), nil
}

func (r *gitRepo) diffCachedNameStatus() ([]byte, error) {
	return r.execInWorkTree("diff", "--cached", "--name-status")
}

func (r *gitRepo) diffBetweenBranches(br1, br2 string) ([]byte, error) {
	return r.execInWorkTree("diff", br1+".."+br2, "--name-status")
}

//supposes output of git branch return branches ordered
func (r *gitRepo) exportChangeSet(branch string) (archive.Archive, error) {
	_, err := r.stash()
	if err != nil {
		return nil, err
	}

	currentBr, err := r.currentBranch()
	if err != nil {
		return nil, err
	}

	_, err = r.checkout(branch)
	if err != nil {
		return nil, err
	}
	defer func() {
		r.checkout(currentBr)
		r.unstash()
	}()

	branches, err := r.branch()
	if err != nil {
		return nil, err
	}
	idx := indexOf(branches, branch)
	switch idx {
	case -1:
		return nil, fmt.Errorf("branch %v not found", branch)
	case 0:
		changes, err := archive.ChangesDirs(r.Path, "")
		if err != nil {
			return nil, err
		}
		return archive.ExportChanges(r.Path, changes)
	default:
		parentBr := branches[idx-1]
		diff, _ := r.diffBetweenBranches(parentBr, branch)
		return exportChanges(r.Path, diff)
	}
}

func exportChanges(rootfs string, diff []byte) (archive.Archive, error) {
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

func (r *gitRepo) execInWorkTree(args ...string) ([]byte, error) {
	args = append([]string{"--git-dir=" + r.Path + "/.git", "--work-tree=" + r.Path}, args...)
	return r.exec(args...)
}

func (r *gitRepo) exec(args ...string) ([]byte, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(gitPath, args...)
	return cmd.CombinedOutput()
}
