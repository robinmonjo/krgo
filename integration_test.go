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
	krgo            = &krgoBin{"krgo"}
	testImages      = []string{"busybox", "progrium/busybox"}
	testV2RegImages = []string{"busybox"}
	privateImage    = "robinmonjo/busybox"
	gitImage        = "busybox:latest"

	minimalLinuxRootfs = []string{"bin", "dev", "etc", "home", "lib", "mnt", "opt", "proc", "root", "run", "sbin", "sys", "tmp", "usr", "var"}
)

type krgoBin struct {
	binary string
}

func (krgo *krgoBin) exec(args ...string) error {
	cmd := exec.Command(krgo.binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error: %v, Out: %s", err, string(out))
	}
	return nil
}

func TestPullImages(t *testing.T) {
	for _, imageName := range testImages {
		rootfs := uniqueStr("rootfs")
		fmt.Printf("Testing %s image ... ", imageName)
		err := krgo.exec("pull", imageName, "-r", rootfs)
		defer os.RemoveAll(rootfs)
		if err != nil {
			t.Fatal(err)
		}
		if err := checkFS(rootfs); err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Ok\n")
	}
}

func TestPullImagesV2(t *testing.T) {
	for _, imageName := range testV2RegImages {
		rootfs := uniqueStr("rootfs")
		fmt.Printf("Testing registry v2 with %s image ... ", imageName)
		err := krgo.exec("pull", imageName, "-r", rootfs, "-v2")
		defer os.RemoveAll(rootfs)
		if err != nil {
			t.Fatal(err)
		}
		if err := checkFS(rootfs); err != nil {
			t.Fatal(err)
		}
		fmt.Printf("Ok\n")
	}
}

func TestPullPrivateImage(t *testing.T) {
	creds := os.Getenv(CREDS_ENV)
	if creds == "" {
		fmt.Printf("Skipping private image test (%S not set)\n", CREDS_ENV)
		return
	}
	fmt.Printf("Testing private %s image ... ", privateImage)
	rootfs := uniqueStr("rootfs")
	err := krgo.exec("pull", privateImage, "-r", rootfs, "-u", creds)
	defer os.RemoveAll(rootfs)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkFS(rootfs); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Ok\n")
}

func TestPullAndPushImage(t *testing.T) {
	//1: download the image
	fmt.Printf("Testing pulling and pushing on image %s\n\tPulling ... ", gitImage)
	rootfs := uniqueStr("rootfs")
	err := krgo.exec("pull", gitImage, "-r", rootfs, "-g")
	defer os.RemoveAll(rootfs)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkFS(rootfs); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Ok\n")

	gitRepo, _ := newGitRepo(rootfs)
	branches, _ := gitRepo.branch()

	fmt.Printf("\tChecking git layering ... ")

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
	fmt.Printf("Ok\n")

	//2: make some modification to it
	creds := os.Getenv(CREDS_ENV)
	if creds == "" {
		fmt.Printf("Skipping push image test (%s not set)\n", CREDS_ENV)
		return
	}

	fmt.Printf("\tModifying and commiting the image ... ")
	newImageName := uniqueStr("robinmonjo/krgo_bb_")

	//make some modifications on the image
	createdFile := "modification.txt"
	deletedFile := "sbin/ifconfig"

	f, err := os.Create(path.Join(rootfs, createdFile))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	err = os.RemoveAll(path.Join(rootfs, deletedFile))
	if err != nil {
		t.Fatal(err)
	}

	//3: commit the image
	err = krgo.exec("commit", "-r", rootfs, "-m", "commit message")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Ok\n")

	//4: push the image
	fmt.Printf("\tPushing %s into %s image ... ", gitImage, newImageName)
	err = krgo.exec("push", newImageName, "-r", rootfs, "-u", creds)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Ok\n")

	//5: donload the image and make sure modifications where applied
	fmt.Printf("\tPulling %s image to make sure modifications were properly applied ... ", newImageName)
	rootfsNew := uniqueStr("rootfs")
	err = krgo.exec("pull", newImageName, "-r", rootfsNew)
	defer os.RemoveAll(rootfsNew)
	if err != nil {
		t.Fatal(err)
	}

	if !fileExists(path.Join(rootfsNew, createdFile)) {
		t.Fatal("expected file %s doesn't exists", createdFile)
	}

	if fileExists(path.Join(rootfsNew, deletedFile)) {
		t.Fatal("file %s should have been deleted", deletedFile)
	}
	fmt.Printf("Ok\n")
}

func checkFS(rootfs string) error {
	for _, file := range minimalLinuxRootfs {
		if !fileExists(path.Join(rootfs, file)) {
			return fmt.Errorf("expected file %s doesn't exists", file)
		}
	}
	return nil
}

func uniqueStr(prefix string) string {
	timestamp := time.Now().Unix()
	timestampStr := strconv.FormatInt(timestamp, 10)
	return prefix + timestampStr
}
