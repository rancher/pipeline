package config

import (
	"github.com/urfave/cli"
)

type config struct {
	CattleUrl       string
	CattleAccessKey string
	CattleSecretKey string
	JenkinsUser     string
	JenkinsToken    string
	JenkinsAddress  string
}

var Config config

func Parse(context *cli.Context) {
	Config.JenkinsAddress = context.String("jenkins_address")
	Config.JenkinsUser = context.String("jenkins_user")
	Config.JenkinsToken = context.String("jenkins_token")
	Config.CattleUrl = context.String("cattle_url")
	Config.CattleAccessKey = context.String("cattle_access_key")
	Config.CattleSecretKey = context.String("cattle_secret_key")
}
