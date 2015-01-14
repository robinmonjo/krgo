package main

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/docker/docker/pkg/archive"
)

const REPO_PATH string = "/tmp/git_repo"

func assertErrNil(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func Test_gitFlow(t *testing.T) {
	r, err := NewGitRepo(REPO_PATH)
	assertErrNil(err, t)

	defer os.RemoveAll(REPO_PATH)

	//Create 3 branches
	for i := 0; i < 3; i++ {
		br := "br" + strconv.Itoa(i)
		_, err = r.CheckoutB(br)
		assertErrNil(err, t)

		curBr, err := r.CurrentBranch()
		assertErrNil(err, t)

		if br != curBr {
			t.Fatalf("current branch: %v expected %v", curBr, br)
		}

		f, err := os.Create(path.Join(r.Path, br+".txt"))
		assertErrNil(err, t)
		f.Close()

		_, err = r.AddAllAndCommit("commit message")
		assertErrNil(err, t)
	}

	exportChangeSet(r, "br0", []string{"br0.txt"}, []string{"br1.txt", "br2.txt", ".git"}, t)
	exportChangeSet(r, "br1", []string{"br1.txt"}, []string{"br0.txt", "br2.txt"}, t)
	exportChangeSet(r, "br2", []string{"br2.txt"}, []string{"br0.txt", "br1.txt"}, t)

	//Modify files
	err = ioutil.WriteFile(path.Join(r.Path, "br0.txt"), []byte("hello world !!"), 0777)
	assertErrNil(err, t)
	_, err = r.AddAllAndCommit("commit message")
	assertErrNil(err, t)
	exportChangeSet(r, "br2", []string{"br2.txt", "br0.txt"}, []string{"br1.txt"}, t)

	//Delete file
	err = os.Remove(path.Join(r.Path, "br1.txt"))
	assertErrNil(err, t)
	_, err = r.AddAllAndCommit("commit message")
	exportChangeSet(r, "br2", []string{"br2.txt", ".wh.br1.txt", "br0.txt"}, []string{"br1.txt"}, t)

	//Uncommited changes
	_, err = r.CheckoutB("br3")
	assertErrNil(err, t)

	f, err := os.Create(path.Join(r.Path, "br3.txt"))
	assertErrNil(err, t)
	f.Close()
	exportUncommitedChangeSet(r, []string{"br3.txt"}, []string{"br1.txt", ".wh.br1.txt", "br0.txt", "br2.txt"}, t)
}

func exportUncommitedChangeSet(r *GitRepo, expectedFiles, unexpectedFiles []string, t *testing.T) {
	tar, err := r.ExportUncommitedChangeSet()
	assertErrNil(err, t)
	defer tar.Close()
	checkTarCorrect(tar, expectedFiles, unexpectedFiles, t)
}

func exportChangeSet(r *GitRepo, branch string, expectedFiles, unexpectedFiles []string, t *testing.T) {
	tar, err := r.ExportChangeSet(branch)
	assertErrNil(err, t)
	defer tar.Close()
	checkTarCorrect(tar, expectedFiles, unexpectedFiles, t)
}

func checkTarCorrect(tar archive.Archive, expectedFiles, unexpectedFiles []string, t *testing.T) {
	err := archive.Untar(tar, "/tmp/tar", nil)
	assertErrNil(err, t)
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
