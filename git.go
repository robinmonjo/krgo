package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"strconv"
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
		if _, err := r.execInWorkTree("config", "user.email", "fake@cargo.com"); err != nil {
			return nil, err
		}
	}

	name, _ := r.userConfig("name")
	if len(name) == 0 {
		if _, err := r.execInWorkTree("config", "user.name", "cargo"); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *gitRepo) userConfig(key string) ([]byte, error) {
	return r.exec("config", "user."+key)
}

func (r *gitRepo) checkout(br branch) ([]byte, error) {
	return r.execInWorkTree("checkout", br.string())
}

func (r *gitRepo) checkoutB(br branch) ([]byte, error) {
	return r.execInWorkTree("checkout", "-b", br.string())
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

func (r *gitRepo) branch() ([]branch, error) {
	b, err := r.execInWorkTree("branch")
	if err != nil {
		return nil, err
	}
	rawBrs := strings.Split(string(b), "\n")
	brs := make([]branch, len(rawBrs))
	for i, br := range rawBrs {
		brs[i] = branch(strings.TrimLeft(br, " *"))
	}
	return brs[:len(brs)-1], nil //remove the last empty line
}

func (r *gitRepo) currentBranch() (branch, error) {
	b, err := r.execInWorkTree("symbolic-ref", "--short", "HEAD")
	return branch(strings.TrimSuffix(string(b), "\n")), err
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

func (r *gitRepo) diff(br1, br2 branch) ([]byte, error) {
	return r.execInWorkTree("diff", br1.string()+".."+br2.string(), "--name-status")
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

func (r *gitRepo) exportChangeSet(br branch) (archive.Archive, error) {
	currentBr, err := r.currentBranch()
	if err != nil {
		return nil, err
	}

	_, err = r.checkout(br)
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

	switch br.number() {
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
		parentBr := branches[br.number()-1]
		diff, _ := r.diff(parentBr, br)
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

//branch specific type for cargo
type branch string

func newBranch(n int, ID string) branch {
	return branch("layer_" + strconv.Itoa(n) + "_" + ID)
}

func (br branch) number() int {
	n, _ := strconv.ParseInt(strings.Split(string(br), "_")[1], 10, 64)
	return int(n)
}

func (br branch) imageID() string {
	return strings.Split(string(br), "_")[2]
}

func (br branch) string() string {
	return string(br)
}
