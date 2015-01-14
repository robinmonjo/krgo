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
	gitImage       string   = "ubuntu:14.04"

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
		"layer1_3b363fd9d7dab4db9591058a3f43e806f6fa6f7e2744b63b2df4b84eadb0685a",
		"layer2_607c5d1cca71dd3b6c04327c3903363079b72ab3e5e4289d74fb00a9ac7ec2aa",
		"layer3_f62feddc05dc67da9b725361f97d7ae72a32e355ce1585f9a60d090289120f73",
		"layer4_8eaa4ff06b53ff7730c4d7a7e21b4426a4b46dee064ca2d5d90d757dc7ea040a",
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
	args := []string{"-r", rootfs, "pull", imageName}
	if credentials != "" {
		args = append([]string{"-u", credentials}, args...)
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
