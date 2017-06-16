package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	"github.com/rancher/pipeline/jenkins"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/restfulserver"
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
		cli.StringFlag{
			Name:   "jenkins_address",
			Usage:  "token of jenkins admin",
			EnvVar: "JENKINS_ADDRESS",
			Value:  "http://jenkins:8080",
		},
		cli.StringFlag{
			Name:   "template_base_path",
			Usage:  "token of jenkins admin",
			EnvVar: "TEMPLATES_BASE_PATH",
			Value:  "/data/rancher-ci/templates",
		},
		cli.StringFlag{
			Name:   "jenkins_config_template",
			Usage:  "Jenkins configuration template file folder",
			EnvVar: "JENKINS_CONFIG_TEMPLATE",
			Value:  "/data/rancher-ci/jenkins",
		},
	}
	app.Run(os.Args)
}

func checkAndRun(c *cli.Context) (rtnerr error) {
	jenkins.InitJenkins(c)
	pipelineContext := pipeline.BuildPipelineContext(c, &jenkins.JenkinsProvider{})
	errChan := make(chan bool)
	go restfulserver.ListenAndServe(pipelineContext, errChan)
	<-errChan
	logrus.Info("Going down")
	return nil
}
