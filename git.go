package main

import (
	"log"
	"os/exec"
)

type GitRepo struct {
	Path string
}

func NewGitRepo(path string) (*GitRepo, error) {
	r := &GitRepo{Path: path}
	err := r.exec("init", path)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *GitRepo) checkout(branch string) error {
	return r.execInWorkTree("checkout", branch)
}

func (r *GitRepo) checkoutB(branch string) error {
	return r.execInWorkTree("checkout", "-b", branch)
}

func (r *GitRepo) add(file string) error {
	return r.execInWorkTree("add", file)
}

func (r *GitRepo) commit(message string) error {
	return r.execInWorkTree("commit", "-m", message)
}

func (r *GitRepo) execInWorkTree(args ...string) error {
	args = append([]string{"--git-dir=" + r.Path + "/.git", "--work-tree=" + r.Path}, args...)
	return r.exec(args...)
}

func (r *GitRepo) exec(args ...string) error {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return err
	}
	cmd := exec.Command(gitPath, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}
