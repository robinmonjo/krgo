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
	pullFlagSet         = flag.NewFlagSet("pull", flag.ExitOnError)
	rootfsDest  *string = pullFlagSet.String("d", "./rootfs", "destination of the resulting rootfs directory")
	credentials *string = pullFlagSet.String("u", "", "docker hub credentials: <username>:<password>")
	gitLayering *bool   = pullFlagSet.Bool("g", false, "use git layering")

	pushFlagSet         = flag.NewFlagSet("push", flag.ExitOnError)
	baseBranch  *string = pushFlagSet.String("br1", "", "base branch")
	newBranch   *string = pushFlagSet.String("br2", "", "new branch")
	rootfs      *string = pushFlagSet.String("d", "", "rootfs path")
)

func init() {
	pullFlagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "PULL:\n  dlrootfs pull <image_name>:[<image_tag>] [-d <rootfs_destination>] [-u <username>:<password>] [-g]\n\n")
		fmt.Fprintf(os.Stderr, "DEFAULT:\n")
		pullFlagSet.PrintDefaults()
	}

	pushFlagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "PUSH:\n  dlrootfs push ...\n\n")
		fmt.Fprintf(os.Stderr, "DEFAULT:\n")
		pushFlagSet.PrintDefaults()
	}
}

func globalUsage() {
	fmt.Fprintf(os.Stderr, "GLOBAL USAGE:\n  dlrootfs pull\n  dlrootfs push\n\n")
	pullFlagSet.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	pushFlagSet.Usage()
}

func main() {

	if len(os.Args) <= 1 {
		globalUsage()
		return
	}

	cmd := os.Args[1]
	subArgs := os.Args[2:]

	switch cmd {
	case "pull":
		pullCmd(subArgs)
	case "push":
		pushCmd(subArgs)
	case "version":
		versionCmd()
	default:
		globalUsage()
	}

}

func versionCmd() {
	fmt.Println(VERSION)
}

func pullCmd(args []string) {
	imageNameTag := args[0]

	pullFlagSet.Parse(args[1:])

	if imageNameTag == "" {
		pullFlagSet.Usage()
		return
	}

	fmt.Printf("Retrieving %v info from the docker hub ...\n", imageNameTag)
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

func pushCmd(args []string) {
	pushFlagSet.Parse(args)
	changes, err := dlrootfs.ExportChanges(*baseBranch, *newBranch, *rootfs)
	if err != nil {
		log.Fatal(err)
	}
	err = dlrootfs.WriteArchiveToFile(changes, "./changes.tar")
	if err != nil {
		log.Fatal(err)
	}
}
