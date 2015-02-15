package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/docker/docker/pkg/archive"
)

const REPO_PATH = "/tmp/git_repo"

var (
	branches = []branch{
		newBranch(0, "4986bf8c15363d1c5d15512d5266f8777bfba4974ac56e3270e7760f6f0a8125"),
		newBranch(1, "4986bf8c15363d1c5d15512d5266f8777bfba4974ac56e3270e7760f6f0a8126"),
		newBranch(2, "4986bf8c15363d1c5d15512d5266f8777bfba4974ac56e3270e7760f6f0a8127"),
		newBranch(3, "4986bf8c15363d1c5d15512d5266f8777bfba4974ac56e3270e7760f6f0a8128"),
	}
)

func asserErrNil(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestGitFlow(t *testing.T) {
	fmt.Printf("Testing git ... ")
	r, err := newGitRepo(REPO_PATH)
	asserErrNil(err, t)

	defer os.RemoveAll(REPO_PATH)

	//Create 3 branches
	for i := 0; i < 3; i++ {
		br := branches[i]
		_, err = r.checkoutB(br)
		asserErrNil(err, t)

		curBr, err := r.currentBranch()
		asserErrNil(err, t)

		if br != curBr {
			t.Fatalf("current branch: %v expected %v", curBr, br)
		}

		f, err := os.Create(path.Join(r.Path, "br"+strconv.Itoa(i)+".txt"))
		asserErrNil(err, t)
		f.Close()

		_, err = r.addAllAndCommit("commit message")
		asserErrNil(err, t)
	}

	exportChangeSet(r, branches[0], []string{"br0.txt"}, []string{"br1.txt", "br2.txt", ".git"}, t)
	exportChangeSet(r, branches[1], []string{"br1.txt"}, []string{"br0.txt", "br2.txt"}, t)
	exportChangeSet(r, branches[2], []string{"br2.txt"}, []string{"br0.txt", "br1.txt"}, t)

	//Modify files
	err = ioutil.WriteFile(path.Join(r.Path, "br0.txt"), []byte("hello world !!"), 0777)
	asserErrNil(err, t)
	_, err = r.addAllAndCommit("commit message")
	asserErrNil(err, t)
	exportChangeSet(r, branches[2], []string{"br2.txt", "br0.txt"}, []string{"br1.txt"}, t)

	//Delete file
	err = os.Remove(path.Join(r.Path, "br1.txt"))
	asserErrNil(err, t)
	_, err = r.addAllAndCommit("commit message")
	exportChangeSet(r, branches[2], []string{"br2.txt", ".wh.br1.txt", "br0.txt"}, []string{"br1.txt"}, t)

	//Uncommited changes
	_, err = r.checkoutB(branches[3])
	asserErrNil(err, t)

	f, err := os.Create(path.Join(r.Path, "br3.txt"))
	asserErrNil(err, t)
	f.Close()
	exportUncommitedChangeSet(r, []string{"br3.txt"}, []string{"br1.txt", ".wh.br1.txt", "br0.txt", "br2.txt"}, t)
	fmt.Printf("OK\n")
}

func exportUncommitedChangeSet(r *gitRepo, expectedFiles, unexpectedFiles []string, t *testing.T) {
	tar, err := r.exportUncommitedChangeSet()
	asserErrNil(err, t)
	defer tar.Close()
	checkTarCorrect(tar, expectedFiles, unexpectedFiles, t)
}

func exportChangeSet(r *gitRepo, br branch, expectedFiles, unexpectedFiles []string, t *testing.T) {
	tar, err := r.exportChangeSet(br)
	asserErrNil(err, t)
	defer tar.Close()
	checkTarCorrect(tar, expectedFiles, unexpectedFiles, t)
}

func checkTarCorrect(tar archive.Archive, expectedFiles, unexpectedFiles []string, t *testing.T) {
	err := archive.Untar(tar, "/tmp/tar", nil)
	asserErrNil(err, t)
	defer os.RemoveAll("/tmp/tar")
	filesShouldExist(true, expectedFiles, "/tmp/tar", t)
	filesShouldExist(false, unexpectedFiles, "/tmp/tar", t)
}

func filesShouldExist(shouldExist bool, files []string, basePath string, t *testing.T) {
	for _, f := range files {
		exist := fileExists(path.Join(basePath, f))
		if exist != shouldExist {
			t.Fatalf("file %v should exist ? %v", f, shouldExist)
		}
	}
}
