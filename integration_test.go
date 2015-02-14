package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"testing"
	"time"
)

const CREDS_ENV string = "DHUB_CREDS"

var (
	cargoBinary  = "cargo"
	testImages   = []string{"busybox", "progrium/busybox"}
	privateImage = "robinmonjo/busybox"
	gitImage     = "busybox:latest"

	rootfs = "tmp_rootfs"

	minimalLinuxRootfs = []string{"bin", "dev", "etc", "home", "lib", "mnt", "opt", "proc", "root", "run", "sbin", "sys", "tmp", "usr", "var", "json"}
)

func itAssertErrNil(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestPullImages(t *testing.T) {
	for _, imageName := range testImages {
		fmt.Printf("Testing %v image ... ", imageName)
		pullImage(imageName, "", false, t, true)
		fmt.Printf("Ok\n")
	}
}

func TestPullPrivateImage(t *testing.T) {
	creds := os.Getenv(CREDS_ENV)
	if creds == "" {
		fmt.Printf("Skipping private image test (%v not set)\n", CREDS_ENV)
		return
	}
	fmt.Printf("Testing private %v image ... ", privateImage)
	pullImage(privateImage, creds, true, t, true)
	fmt.Printf("Ok\n")
}

func TestPullImageWithGit(t *testing.T) {
	fmt.Printf("Testing using git layering %v image ... ", gitImage)
	pullImage(gitImage, "", true, t, false)

	gitRepo, _ := newGitRepo(rootfs)
	branches, _ := gitRepo.branch()

	fmt.Printf("Checking git layering ... ")

	expectedbranches := []string{
		"layer_0_511136ea3c5a64f264b78b5433614aec563103b4d4702f3ba7d4d2698e22c158",
		"layer_1_df7546f9f060a2268024c8a230d8639878585defcc1bc6f79d2728a13957871b",
		"layer_2_ea13149945cb6b1e746bf28032f02e9b5a793523481a0a18645fc77ad53c4ea2",
		"layer_3_4986bf8c15363d1c5d15512d5266f8777bfba4974ac56e3270e7760f6f0a8125",
	}

	for i, br := range branches {
		if br.string() != expectedbranches[i] {
			t.Fatal("Expected branch", expectedbranches[i], "got", br)
		}
	}
	fmt.Printf("OK\n")
}

func TestPushImage(t *testing.T) {
	//pushing the previously downloaded image into a random folder
	defer os.RemoveAll(rootfs)
	creds := os.Getenv(CREDS_ENV)
	if creds == "" {
		fmt.Printf("Skipping push image test (%v not set)\n", CREDS_ENV)
		return
	}

	timestamp := time.Now().Unix()
	timestampStr := strconv.FormatInt(timestamp, 10)
	newImageNameTag := "robinmonjo/cargo_bb_" + timestampStr + ":testing"

	fmt.Printf("Testing push image %v ... ", newImageNameTag)
	//make some modifications on the image
	f, err := os.Create(path.Join(rootfs, "modification.txt"))
	itAssertErrNil(err, t)
	f.Close()

	//commit the image
	commitImage("commit message", t)
	fmt.Printf("commit ok ... ")

	//push it
	pushImage(newImageNameTag, creds, t)
}

//helpers
func pullImage(imageNameTag, credentials string, gitLayering bool, t *testing.T, cleanup bool) {
	if cleanup {
		defer os.RemoveAll(rootfs)
	}
	args := []string{"pull", imageNameTag, "-r", rootfs}
	if credentials != "" {
		args = append(args, []string{"-u", credentials}...)
	}
	if gitLayering {
		args = append(args, "-g")
	}

	cmd := exec.Command(cargoBinary, args...)
	err := cmd.Start()
	itAssertErrNil(err, t)

	err = cmd.Wait()
	itAssertErrNil(err, t)

	fmt.Printf("Checking FS ... ")
	for _, file := range minimalLinuxRootfs {
		if !fileExists(path.Join(rootfs, file)) {
			t.Fatalf("expected file %v doesn't exists\n", file)
		}
	}
}

func pushImage(imageNameTag, credentials string, t *testing.T) {
	cmd := exec.Command(cargoBinary, "push", imageNameTag, "-r", rootfs, "-u", credentials)
	err := cmd.Start()
	itAssertErrNil(err, t)
	err = cmd.Wait()
	itAssertErrNil(err, t)
}

func commitImage(message string, t *testing.T) {
	cmd := exec.Command(cargoBinary, "commit", "-r", rootfs, "-m", message)
	err := cmd.Start()
	itAssertErrNil(err, t)
	err = cmd.Wait()
	itAssertErrNil(err, t)
}
