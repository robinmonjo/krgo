package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/rmonjo/dlrootfs"
)

const CREDS_ENV string = "DLROOTFS_CREDS"

var (
	dlrootfsBinary string   = "dlrootfs"
	testImages     []string = []string{"busybox", "progrium/busybox"}
	unknownImage   string   = "unknownZRTFGHUIJKLMOPRST"
	privateImage   string   = "robinmonjo/debian"
	gitImage       string   = "ubuntu:14.04"

	minimalLinuxRootDirs []string = []string{"bin", "dev", "etc", "home", "lib", "mnt", "opt", "proc", "root", "run", "sbin", "sys", "tmp", "usr", "var"}
)

func assertErrNil(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func assertErrNotNil(err error, t *testing.T) {
	if err == nil {
		t.Fatal(err)
	}
}

func Test_downloadImage(t *testing.T) {
	fmt.Printf("Testing unknown %v image ... ", unknownImage)
	downloadImage(unknownImage, "tmp_unknown", "", false, assertErrNotNil, t)
	fmt.Printf("Ok\n")

	for _, imageName := range testImages {
		fmt.Printf("Testing %v image ... ", imageName)
		downloadImage(imageName, "tmp_test", "", true, assertErrNil, t)
		fmt.Printf("Ok\n")
	}
}

func Test_downloadPrivateImage(t *testing.T) {
	creds := os.Getenv(CREDS_ENV)
	if creds == "" {
		fmt.Printf("Skipping private image test (%v not set)\n", CREDS_ENV)
		return
	}
	fmt.Printf("Testing private %v image ... ", privateImage)
	downloadImage(privateImage, "tmp_priv_test", creds, true, assertErrNil, t)
	fmt.Printf("Ok\n")
}

func downloadImage(imageName, rootfsDest, credentials string, checkFs bool, assert func(error, *testing.T), t *testing.T) {
	defer os.RemoveAll(rootfsDest)

	cmd := exec.Command(dlrootfsBinary, "-i", imageName, "-d", rootfsDest, "-u", credentials)
	err := cmd.Start()
	assertErrNil(err, t)

	err = cmd.Wait()
	assert(err, t)

	if !checkFs {
		return
	}

	fmt.Printf("Checking FS ... ")
	for _, dir := range minimalLinuxRootDirs {
		checkDirExists(rootfsDest+"/"+dir, t)
	}

}

func checkDirExists(dir string, t *testing.T) {
	src, err := os.Stat(dir)
	assertErrNil(err, t)

	if !src.IsDir() {
		t.Fatal(dir, "not a directory")
	}
}

func Test_downloadWithGitLayers(t *testing.T) {
	fmt.Printf("Testing git layering ... ")
	rootfsDest := "./ubuntu"
	defer os.RemoveAll(rootfsDest)
	cmd := exec.Command(dlrootfsBinary, "-i", gitImage, "-d", rootfsDest, "-g")
	err := cmd.Start()
	assertErrNil(err, t)

	err = cmd.Wait()
	assertErrNil(err, t)

	gitRepo, _ := dlrootfs.NewGitRepo(rootfsDest)
	out, _ := gitRepo.Branch()

	expectedBranches := []string{
		"layer0_511136ea3c5a",
		"layer1_01bf15a18638",
		"layer2_30541f8f3062",
		"layer3_e1cdf371fbde",
		"* layer4_9bd07e480c5b",
	}

	branches := strings.Split(string(out), "\n")

	for i, branch := range branches {
		trimmedBranch := strings.Trim(branch, " \n")
		if trimmedBranch == "" {
			continue
		}
		expectedBranch := expectedBranches[i]
		if trimmedBranch != expectedBranch {
			t.Fatal("Expected branch", expectedBranch, " got ", trimmedBranch)
		}
	}
	fmt.Printf("OK\n")
}
