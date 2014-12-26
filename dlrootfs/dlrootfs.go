package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rmonjo/dlrootfs"
)

const VERSION string = "1.4.0"

var (
	flagset = flag.NewFlagSet("dlrootfs", flag.ExitOnError)

	rootfsDest  *string = flagset.String("d", "./rootfs", "destination of the resulting rootfs directory")
	credentials *string = flagset.String("u", "", "docker hub credentials: <username>:<password>")
	gitLayering *bool   = flagset.Bool("g", false, "use git layering")
)

func init() {
	flagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dlrootfs <image_name>:[<image_tag>] [-d <rootfs_destination>] [-u <username>:<password>] [-g]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs ubuntu  #if no tag, use latest\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs ubuntu:precise -d ubuntu_rootfs\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs dockefile/elasticsearch:latest\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs my_repo/my_image:latest -u username:password\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs version\n")
		fmt.Fprintf(os.Stderr, "Default:\n")
		flagset.PrintDefaults()
	}
}

func main() {

	if len(os.Args) <= 1 {
		flagset.Usage()
		return
	}

	imageNameTag := os.Args[1]

	switch imageNameTag {
	case "version":
		fmt.Println(VERSION)
	case "":
		flagset.Usage()
	default:
		pullImage(imageNameTag, os.Args[2:])
	}
}

func pullImage(imageNameTag string, args []string) {
	flagset.Parse(args)

	imageName, imageTag := dlrootfs.ParseImageNameTag(imageNameTag)
	userName, password := dlrootfs.ParseCredentials(*credentials)

	fmt.Printf("Opening a session for %v ...\n", imageName)
	session, err := dlrootfs.NewHubSession(imageName, userName, password)
	if err != nil {
		log.Fatal(err)
	}

	err = session.DownloadFlattenedImage(imageName, imageTag, *rootfsDest, *gitLayering, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nRootfs of %v:%v in %v\n", imageName, imageTag, *rootfsDest)
	if *credentials != "" {
		fmt.Printf("WARNING: don't forget to remove your docker hub credentials from your history !!\n")
	}
}
