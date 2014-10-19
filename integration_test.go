package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
)

var (
	dlrootfsBinary string   = "dlrootfs"
	testImages     []string = []string{"busybox", "pogrium/busybox"}
	scratchImage   string   = "scratch"
	unknownImage   string   = "unknownZRTFGHUIJKLMOPRST"

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
	downloadImage(unknownImage, "tmp_unknown", false, assertErrNotNil, t)
	fmt.Printf("Ok\n")

	if runtime.GOOS != "linux" {
		fmt.Printf("Warning, just testing the %v image on %v \n Testing %v image ... ", scratchImage, scratchImage, runtime.GOOS)
		downloadImage(scratchImage, "tmp_scratch", false, assertErrNil, t)
		fmt.Printf("Ok\n")
		return
	}

	for _, imageName := range testImages {
		fmt.Printf("Testing %v image ... ", imageName)
		downloadImage(imageName, "tmp_test", true, assertErrNil, t)
		fmt.Printf("Ok\n")
	}

}

func downloadImage(imageName, rootfsDest string, checkFs bool, assert func(error, *testing.T), t *testing.T) {
	defer os.Remove(rootfsDest)

	cmd := exec.Command(dlrootfsBinary, "-i", imageName, "-d", rootfsDest)
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
