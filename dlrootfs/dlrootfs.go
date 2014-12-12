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
	globalFlagset = flag.NewFlagSet("dlrootfs", flag.ExitOnError)

	rootfsDest  *string = globalFlagset.String("d", "./rootfs", "destination of the resulting rootfs directory")
	credentials *string = globalFlagset.String("u", "", "docker hub credentials: <username>:<password>")
	gitLayering *bool   = globalFlagset.Bool("g", false, "use git layering")
)

func init() {
	globalFlagset.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dlrootfs -i <image_name>:[<image_tag>] [-d <rootfs_destination>] [-u <username>:<password>] [-g]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs -i ubuntu  #if no tag, use latest\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs -i ubuntu:precise -d ubuntu_rootfs\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs -i dockefile/elasticsearch:latest\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs -i my_repo/my_image:latest -u username:password\n")
		fmt.Fprintf(os.Stderr, "  dlrootfs version\n")
		fmt.Fprintf(os.Stderr, "Default:\n")
		globalFlagset.PrintDefaults()
	}
}

func main() {

	if len(os.Args) <= 1 {
		globalFlagset.Usage()
		return
	}

	imageNameTag := os.Args[1]

	if imageNameTag == "version" {
		fmt.Println(VERSION)
		return
	}

	globalFlagset.Parse(os.Args[2:])

	if imageNameTag == "" {
		globalFlagset.Usage()
		return
	}

	fmt.Printf("Retrieving %v info from the DockerHub ...\n", imageNameTag)
	pullContext, err := dlrootfs.RequestPullContext(imageNameTag, *credentials)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Image ID: %v\n", pullContext.ImageId)

	err = dlrootfs.DownloadImage(pullContext, *rootfsDest, *gitLayering, true)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nRootfs of %v:%v in %v\n", pullContext.ImageName, pullContext.ImageTag, *rootfsDest)
	if *credentials != "" {
		fmt.Printf("WARNING: don't forget to remove your docker hub credentials from your history !!\n")
	}
}
