package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	"github.com/urfave/cli"
)

var VERSION = "v0.0.0-dev"

func main() {
	app := cli.NewApp()
	app.Name = "pipeline"
	app.Version = VERSION
	app.Usage = "You need help!"
	app.Action = checkAndRun
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "jenkins_user",
			Usage:  "User of jenkins admin",
			EnvVar: "JENKINS_USER",
		},
		cli.StringFlag{
			Name:   "jenkins_token",
			Usage:  "token of jenkins admin",
			EnvVar: "JENKINS_TOKEN",
		},
	}
	app.Run(os.Args)
}

func checkAndRun(c *cli.Context) (rtnerr error) {
	defer func() {
		if rtnerr != nil {
			logrus.Fatal(rtnerr)
		}
	}()
	if err := c.GlobalSet("jenkins_user", c.String("jenkins_user")); err != nil {
		return err
	}
	c.GlobalSet("jenkins_user", c.String("jenkins_user"))
	logrus.Info("get in")
	return nil
}
