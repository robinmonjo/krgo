package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/rmonjo/dlrootfs"
)

const VERSION string = "1.4.0"

var Commands = []cli.Command{
	pullCmd,
	pushCmd,
}

var pullCmd = cli.Command{
	Name:        "pull",
	Usage:       "pull <image_name>:[<image_tag>] [-g]",
	Description: "pull an image from the dockerhub. Image name format is reposiroty/name:tag",
	Action:      pull,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "g, git-layering",
			Usage: "Download image in a git repository where each layer is a branch. Needed if you want to push your image afterward",
		},
	},
}

var pushCmd = cli.Command{
	Name:        "push",
	Usage:       "push <image_name>:[<image_tag>] [-m <comment>]",
	Description: "push an image on the dockerhub. Image name format is repository/name:tag",
	Action:      push,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "m, message",
			Usage: "Message to indicates you changes",
			Value: "dlrootfs push",
		},
	},
}

func main() {

	app := cli.NewApp()
	app.Name = "dlrootfs"
	app.Version = VERSION
	app.Usage = "docker hub without docker"
	app.Author = "Robin Monjo"
	app.Email = "robinmonjo@gmail.com"
	app.Commands = Commands

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "u, user",
			Usage: "dockerhub credentials with format <username>:<password>",
		},
		cli.StringFlag{
			Name:  "r, rootfs",
			Usage: "path where to store the rootfs (pull) or where to find the rootfs (push)",
			Value: "./rootfs",
		},
	}

	dlrootfs.PrintOutput = true

	app.Run(os.Args)
}

func pull(c *cli.Context) {
	if len(c.Args()) == 0 {
		cli.ShowSubcommandHelp(c)
		return
	}

	imageName, imageTag := dlrootfs.ParseImageNameTag(c.Args()[0])
	userName, password := dlrootfs.ParseCredentials(c.GlobalString("user"))

	fmt.Printf("Opening a session for %v ...\n", imageName)
	session, err := dlrootfs.NewHubSession(imageName, userName, password)
	if err != nil {
		log.Fatal(err)
	}

	err = session.DownloadFlattenedImage(imageName, imageTag, c.GlobalString("rootfs"), c.Bool("git-layering"))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nRootfs of %v:%v in %v\n", imageName, imageTag, c.GlobalString("rootfs"))
}

func push(c *cli.Context) {
	if len(c.Args()) == 0 {
		cli.ShowSubcommandHelp(c)
		return
	}

	imageName, imageTag := dlrootfs.ParseImageNameTag(c.Args()[0])
	userName, password := dlrootfs.ParseCredentials(c.GlobalString("user"))

	fmt.Printf("Opening a session for %v ...\n", imageName)
	session, err := dlrootfs.NewHubSession(imageName, userName, password)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Extracting changes\n")
	changes, err := dlrootfs.ExportChanges(c.GlobalString("rootfs"))
	if err != nil {
		log.Fatal(err)
	}

	err = session.PushImageLayer(changes, imageName, imageTag, c.String("message"), c.GlobalString("rootfs"))
	if err != nil {
		log.Fatal(err)
	}
}
