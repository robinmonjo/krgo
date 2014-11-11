package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
)

const CREDS_ENV string = "DLROOTFS_CREDS"

var (
	dlrootfsBinary string   = "dlrootfs"
	testImages     []string = []string{"busybox", "progrium/busybox", "debian"}
	unknownImage   string   = "unknownZRTFGHUIJKLMOPRST"
	privateImage   string   = "robinmonjo/debian"

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
	fmt.Printf("Testing %v image ... ", privateImage)
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
