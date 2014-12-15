package dlrootfs

import (
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

func NewGitRepo(path string) (*GitRepo, error) {
	r := &GitRepo{Path: path}

	if _, err := os.Stat(path + "/.git"); err == nil {
		return r, nil
	}

	_, err := r.exec("init", path)
	if err != nil {
		return nil, err
	}
	_, err = r.execInWorkTree("config", "user.email", "mail@mail.com")
	if err != nil {
		return nil, err
	}

	_, err = r.execInWorkTree("config", "user.name", "dlrootfs")
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *GitRepo) Checkout(branch string) ([]byte, error) {
	return r.execInWorkTree("checkout", branch)
}

func (r *GitRepo) CheckoutB(branch string) ([]byte, error) {
	return r.execInWorkTree("checkout", "-b", branch)
}

func (r *GitRepo) Add(file string) ([]byte, error) {
	return r.execInWorkTree("add", file)
}

func (r *GitRepo) Commit(message string) ([]byte, error) {
	return r.execInWorkTree("commit", "-m", message)
}

func (r *GitRepo) Branch() ([]byte, error) {
	return r.execInWorkTree("branch")
}

func (r *GitRepo) DiffStatusName(br1, br2 string) ([]byte, error) {
	return r.execInWorkTree("diff", br1+".."+br2, "--name-status")
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
