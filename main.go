package main

import (
	"fmt"
	"log"
	"os"

	"github.com/codegangsta/cli"
)

const VERSION string = "1.4.1"

var (
	//shared flags
	userFlag   = cli.StringFlag{Name: "u, user", Usage: "dockerhub credentials (format: username:password)"}
	rootfsFlag = cli.StringFlag{Name: "r, rootfs", Usage: "path of the root FS (default: rootfs)", Value: "rootfs"}

	//commands
	pullCmd = cli.Command{
		Name:        "pull",
		Usage:       "pull an image",
		Description: "pull image [-r rootfs] [-u user] [-g]",
		Action:      pull,
		Flags: []cli.Flag{
			cli.BoolFlag{Name: "g, git-layering", Usage: "use git layering (needed to push afteward)"},
			userFlag,
			rootfsFlag,
		},
	}

	pushCmd = cli.Command{
		Name:        "push",
		Usage:       "push an image",
		Description: "push image [-r rootfs] -u user",
		Action:      push,
		Flags: []cli.Flag{
			userFlag,
			rootfsFlag,
		},
	}

	commitCmd = cli.Command{
		Name:        "commit",
		Usage:       "commit changes to an image pulled with -g",
		Description: "commit [-r rootfs] -m message",
		Action:      commit,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "m, message", Usage: "commit message"},
			rootfsFlag,
		},
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "cargo"
	app.Version = VERSION
	app.Usage = "docker hub without docker"
	app.Author = "Robin Monjo"
	app.Email = "robinmonjo@gmail.com"
	app.Commands = []cli.Command{pullCmd, pushCmd, commitCmd}

	app.Run(os.Args)
}

func pull(c *cli.Context) {
	imageName, imageTag := parseImageNameTag(c.Args().First())
	userName, password := parseCredentials(c.String("user"))

	fmt.Printf("Pulling image %v:%v ...\n", imageName, imageTag)
	session, err := newHubSession(imageName, userName, password)
	if err != nil {
		log.Fatal(err)
	}

	if c.Bool("git-layering") {
		err = session.pullRepository(imageName, imageTag, c.String("rootfs"))
	} else {
		err = session.pullImage(imageName, imageTag, c.String("rootfs"))
	}
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done. Rootfs of %v:%v in %v\n", imageName, imageTag, c.String("rootfs"))
}

func commit(c *cli.Context) {
	err := commitChanges(c.String("rootfs"), c.String("message"))
	if err != nil {
		log.Fatalf("Something went wrong: %v\nGit repo may have been altered. Please make sure it's fine before commiting again\n", err)
	}
	fmt.Printf("Done\n")
}

func push(c *cli.Context) {
	imageName, imageTag := parseImageNameTag(c.Args().First())
	userName, password := parseCredentials(c.String("user"))

	fmt.Printf("Pushing image %v:%v ...\n", imageName, imageTag)
	session, err := newHubSession(imageName, userName, password)
	if err != nil {
		log.Fatal(err)
	}

	err = session.pushRepository(imageName, imageTag, c.String("rootfs"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Done: https://registry.hub.docker.com/u/%v\n", imageName)
}
