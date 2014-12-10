package main

import (
	"os"
	"os/exec"
)

type GitRepo struct {
	Path string
}

func NewGitRepo(path string) (*GitRepo, error) {
	r := &GitRepo{Path: path}

	// equivalent to Python's `if os.path.exists(filename)`
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

func (r *GitRepo) checkout(branch string) ([]byte, error) {
	return r.execInWorkTree("checkout", branch)
}

func (r *GitRepo) checkoutB(branch string) ([]byte, error) {
	return r.execInWorkTree("checkout", "-b", branch)
}

func (r *GitRepo) add(file string) ([]byte, error) {
	return r.execInWorkTree("add", file)
}

func (r *GitRepo) commit(message string) ([]byte, error) {
	return r.execInWorkTree("commit", "-m", message)
}

func (r *GitRepo) branch() ([]byte, error) {
	return r.execInWorkTree("branch")
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
