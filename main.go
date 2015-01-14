package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
)

const VERSION string = "1.4.0"

var Commands = []cli.Command{
	pullCmd,
	pushCmd,
	commitCmd,
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
	Usage:       "push <image_name>:[<image_tag>]",
	Description: "push an image on the dockerhub. Image name format is repository/name:tag",
	Action:      push,
}

var commitCmd = cli.Command{
	Name:        "commit",
	Usage:       "commit [-m <commit message>]",
	Description: "commit the changes to prepare a push",
	Action:      commit,
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
			Usage: "path where to store the rootfs (pull) or where to find the rootfs (push) (default: ./rootfs)",
			Value: "./rootfs",
		},
	}

	app.Run(os.Args)
}

func pull(c *cli.Context) {
	imageName, imageTag := ParseImageNameTag(c.Args().First())
	userName, password := ParseCredentials(c.GlobalString("user"))

	fmt.Printf("Opening a session for %v ...\n", imageName)
	session, err := NewHubSession(imageName, userName, password)
	if err != nil {
		log.Fatal(err)
	}

	if c.Bool("git-layering") {
		err = session.PullRepository(imageName, imageTag, c.GlobalString("rootfs"))
	} else {
		err = session.PullImage(imageName, imageTag, c.GlobalString("rootfs"))
	}
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nRootfs of %v:%v in %v\n", imageName, imageTag, c.GlobalString("rootfs"))
}

func commit(c *cli.Context) {
	err := CommitChanges(c.GlobalString("rootfs"), c.String("message"))
	if err != nil {
		log.Fatal(err)
	}
}

func push(c *cli.Context) {
	imageName, imageTag := ParseImageNameTag(c.Args().First())
	userName, password := ParseCredentials(c.GlobalString("user"))

	fmt.Printf("Opening a session for %v ...\n", imageName)
	session, err := NewHubSession(imageName, userName, password)
	if err != nil {
		log.Fatal(err)
	}

	err = session.PushRepository(imageName, imageTag, c.GlobalString("rootfs"))
	if err != nil {
		log.Fatal(err)
	}

}
