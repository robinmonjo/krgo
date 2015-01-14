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
	DIFF_ADDED    string = "A"
	DIFF_MODIFIED string = "M"
	DIFF_DELETED  string = "D"
)

type GitRepo struct {
	Path string
}

func IsGitRepo(repoPath string) bool {
	return fileExists(path.Join(repoPath, ".git"))
}

func NewGitRepo(path string) (*GitRepo, error) {
	r := &GitRepo{Path: path}

	if IsGitRepo(r.Path) {
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

func (r *GitRepo) userConfig(key string) ([]byte, error) {
	return r.exec("config", "user."+key)
}

func (r *GitRepo) Checkout(branch string) ([]byte, error) {
	return r.execInWorkTree("checkout", branch)
}

func (r *GitRepo) CheckoutB(branch string) ([]byte, error) {
	return r.execInWorkTree("checkout", "-b", branch)
}

func (r *GitRepo) AddAllAndCommit(message string) ([]byte, error) {
	bAdd, err := r.Add(".")
	if err != nil {
		return bAdd, err
	}
	bCi, err := r.Commit(message)
	return append(bAdd, bCi...), err
}

func (r *GitRepo) Add(file string) ([]byte, error) {
	return r.execInWorkTree("add", file, "--all")
}

func (r *GitRepo) Commit(message string) ([]byte, error) {
	return r.execInWorkTree("commit", "-m", message)
}

func (r *GitRepo) Branch() ([]string, error) {
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

func (r *GitRepo) CurrentBranch() (string, error) {
	b, err := r.execInWorkTree("symbolic-ref", "--short", "HEAD")
	return strings.TrimSuffix(string(b), "\n"), err
}

func (r *GitRepo) CountBranches() (int, error) {
	branches, err := r.Branch()
	if err != nil {
		return -1, err
	}
	return len(branches), nil
}

func (r *GitRepo) DiffCachedNameStatus() ([]byte, error) {
	return r.execInWorkTree("diff", "--cached", "--name-status")
}

func (r *GitRepo) DiffBetweenBranches(br1, br2 string) ([]byte, error) {
	return r.execInWorkTree("diff", br1+".."+br2, "--name-status")
}

//export every uncommited changes in the current branch
func (r *GitRepo) ExportUncommitedChangeSet() (archive.Archive, error) {
	r.Add(".")

	diff, err := r.DiffCachedNameStatus()
	if err != nil {
		return nil, err
	}
	return exportChanges(r.Path, diff)
}

//assumes output of git branch returns branches ordered
func (r *GitRepo) ExportChangeSet(branch string) (archive.Archive, error) {
	currentBr, err := r.CurrentBranch()
	if err != nil {
		return nil, err
	}

	_, err = r.Checkout(branch)
	if err != nil {
		return nil, err
	}

	defer func() {
		r.Checkout(currentBr)
	}()

	branches, err := r.Branch()
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
		var curatedChanges []archive.Change
		for _, ch := range changes {
			if !strings.HasPrefix(ch.Path, "/.git") {
				curatedChanges = append(curatedChanges, ch)
			}
		}
		return archive.ExportChanges(r.Path, curatedChanges)
	default:
		parentBr := branches[idx-1]
		diff, _ := r.DiffBetweenBranches(parentBr, branch)
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

func (r *GitRepo) execInWorkTree(args ...string) ([]byte, error) {
	args = append([]string{"--git-dir=" + r.Path + "/.git", "--work-tree=" + r.Path}, args...)
	return r.exec(args...)
}

func (r *GitRepo) exec(args ...string) ([]byte, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(gitPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("cmd %v failed with output %v (%v)", cmd, string(out), err)
	}
	return out, nil
}
