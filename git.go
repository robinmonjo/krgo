package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/docker/docker/pkg/archive"
)

const (
	DIFF_ADDED    = "A"
	DIFF_MODIFIED = "M"
	DIFF_DELETED  = "D"
)

type gitRepo struct {
	Path string
}

func isGitRepo(repoPath string) bool {
	return fileExists(path.Join(repoPath, ".git"))
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
		_, err = r.execInWorkTree("config", "user.email", "fake@cargo.com")
		if err != nil {
			return nil, err
		}
	}

	name, _ := r.userConfig("name")
	if len(name) == 0 {
		_, err = r.execInWorkTree("config", "user.name", "cargo")
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
	if err := validateBranch(branch); err != nil {
		return nil, err
	}
	return r.execInWorkTree("checkout", "-b", branch)
}

func (r *gitRepo) addAllAndCommit(message string) ([]byte, error) {
	badd, err := r.add(".")
	if err != nil {
		return badd, err
	}
	bCi, err := r.commit(message)
	return append(badd, bCi...), err
}

func (r *gitRepo) add(file string) ([]byte, error) {
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
	for i, br := range brs {
		brs[i] = strings.TrimLeft(br, " *")
	}
	return brs[:len(brs)-1], nil //remove the last empty line
}

func (r *gitRepo) currentBranch() (string, error) {
	b, err := r.execInWorkTree("symbolic-ref", "--short", "HEAD")
	return strings.TrimSuffix(string(b), "\n"), err
}

func (r *gitRepo) countBranch() (int, error) {
	branches, err := r.branch()
	if err != nil {
		return -1, err
	}
	return len(branches), nil
}

func (r *gitRepo) diffCached() ([]byte, error) {
	return r.execInWorkTree("diff", "--cached", "--name-status")
}

func (r *gitRepo) diff(br1, br2 string) ([]byte, error) {
	return r.execInWorkTree("diff", br1+".."+br2, "--name-status")
}

//export every uncommited changes in the current branch
func (r *gitRepo) exportUncommitedChangeSet() (archive.Archive, error) {
	r.add(".")

	diff, err := r.diffCached()
	if err != nil {
		return nil, err
	}
	return exportChanges(r.Path, diff)
}

//assumes output of git branch returns branches ordered
func (r *gitRepo) exportChangeSet(branch string) (archive.Archive, error) {
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
	}()

	branches, err := r.branch()
	if err != nil {
		return nil, err
	}

	idx, err := exportLayerNumberFromBranch(branch)
	if err != nil {
		return nil, err
	}

	switch idx {
	case 0:
		changes, err := archive.ChangesDirs(r.Path, "")
		if err != nil {
			return nil, err
		}
		var curatedChanges []archive.Change
		for _, ch := range changes {
			if !strings.HasPrefix(ch.Path, "/.git") {
				curatedChanges = append(curatedChanges, ch)
			}
		}
		return archive.ExportChanges(r.Path, curatedChanges)
	default:
		parentBr := branches[idx-1]
		diff, _ := r.diff(parentBr, branch)
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
	args = append([]string{"--git-dir=" + path.Join(r.Path, "/.git"), "--work-tree=" + r.Path}, args...)
	return r.exec(args...)
}

func (r *gitRepo) exec(args ...string) ([]byte, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(gitPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%v (%v)", string(out), err)
	}
	return out, nil
}
