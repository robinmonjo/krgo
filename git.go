package dlrootfs

import (
	"bytes"
	"os"
	"os/exec"
)

const (
	DIFF_ADDED    string = "A"
	DIFF_MODIFIED string = "M"
	DIFF_DELETED  string = "D"
)

type GitRepo struct {
	Path string
}

func IsGitRepo(path string) bool {
	_, err := os.Stat(path + "/.git")
	return err == nil
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

func (r *GitRepo) AddAllAndCommit(file, message string) ([]byte, error) {
	bAdd, err := r.AddAll(file)
	if err != nil {
		return bAdd, err
	}
	bCi, err := r.Commit(message)
	return append(bAdd, bCi...), err
}

func (r *GitRepo) AddAll(file string) ([]byte, error) {
	return r.execInWorkTree("add", file, "--all")
}

func (r *GitRepo) Commit(message string) ([]byte, error) {
	return r.execInWorkTree("commit", "-m", message)
}

func (r *GitRepo) Branch() ([]byte, error) {
	return r.execInWorkTree("branch")
}

func (r *GitRepo) CountBranches() (int, error) {
	b, err := r.Branch()
	if err != nil {
		return -1, err
	}
	return len(bytes.Split(b, []byte("\n"))), nil
}

func (r *GitRepo) DiffCachedNameStatus() ([]byte, error) {
	return r.execInWorkTree("diff", "--cached", "--name-status")
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
	return cmd.CombinedOutput()
}
