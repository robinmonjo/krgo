package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
)

const CREDS_ENV string = "DLROOTFS_CREDS"

var (
	dlrootfsBinary string   = "dlrootfs"
	testImages     []string = []string{"busybox", "progrium/busybox"}
	privateImage   string   = "robinmonjo/debian"
	gitImage       string   = "busybox:latest"

	rootfs string = "tmp_rootfs"

	minimalLinuxRootfs []string = []string{"bin", "dev", "etc", "home", "lib", "mnt", "opt", "proc", "root", "run", "sbin", "sys", "tmp", "usr", "var", "json"}
)

func Test_pullImages(t *testing.T) {
	for _, imageName := range testImages {
		fmt.Printf("Testing %v image ... ", imageName)
		pullImage(imageName, "", false, t, true)
		fmt.Printf("Ok\n")
	}
}

func Test_pullPrivateImage(t *testing.T) {
	creds := os.Getenv(CREDS_ENV)
	if creds == "" {
		fmt.Printf("Skipping private image test (%v not set)\n", CREDS_ENV)
		return
	}
	fmt.Printf("Testing private %v image ... ", privateImage)
	pullImage(privateImage, creds, true, t, true)
	fmt.Printf("Ok\n")
}

func Test_pullImageWithGit(t *testing.T) {
	fmt.Printf("Testing using git layering %v image ... ", gitImage)
	defer os.RemoveAll(rootfs)
	pullImage(gitImage, "", true, t, false)

	gitRepo, _ := NewGitRepo(rootfs)
	branches, _ := gitRepo.Branch()

	fmt.Printf("Checking git layering ... ")

	expectedBranches := []string{
		"layer0_511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158",
		"layer1_df7546f9f060a2268024c8a230d8639878585defcc1bc6f79d2728a13957871b",
		"layer2_ea13149945cb6b1e746bf28032f02e9b5a793523481a0a18645fc77ad53c4ea2",
		"layer3_4986bf8c15363d1c5d15512d5266f8777bfba4974ac56e3270e7760f6f0a8125",
	}

	for i, branch := range branches {
		if branch != expectedBranches[i] {
			t.Fatal("Expected branch", expectedBranches[i], "got", branch)
		}
	}
	fmt.Printf("OK\n")
}

func pullImage(imageName, credentials string, gitLayering bool, t *testing.T, cleanup bool) {
	if cleanup {
		defer os.RemoveAll(rootfs)
	}
	args := []string{"pull", imageName, "-r", rootfs}
	if credentials != "" {
		args = append(args, []string{"-u", credentials}...)
	}
	if gitLayering {
		args = append(args, "-g")
	}

	cmd := exec.Command(dlrootfsBinary, args...)
	err := cmd.Start()
	assertErrNil(err, t)

	err = cmd.Wait()
	assertErrNil(err, t)

	fmt.Printf("Checking FS ... ")
	for _, file := range minimalLinuxRootfs {
		if !fileExists(path.Join(rootfs, file)) {
			t.Fatalf("expected file %v doesn't exists\n", file)
		}
	}
}
