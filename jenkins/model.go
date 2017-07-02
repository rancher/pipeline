package jenkins

import "encoding/xml"

const (
	GIT_SCM_CLASS      = "hudson.plugins.git.GitSCM"
	GIT_SCM_PLUGIN     = "git@3.3.1"
	SCM_CONFIG_VERSION = 2
)

type JenkinsProject struct {
	XMLName                          xml.Name `xml:"project"`
	Actions                          string   `xml:"actions"`
	Description                      string   `xml:"description"`
	KeepDependencies                 bool     `xml:"keepDependencies"`
	Properties                       string
	Scm                              JenkinsSCM       `xml:"scm"`
	CanRoam                          bool             `xml:"canRoam"`
	Disabled                         bool             `xml:"disabled"`
	BlockBuildWhenDownstreamBuilding bool             `xml:"blockBuildWhenDownstreamBuilding"`
	BlockBuildWhenUpstreamBuilding   bool             `xml:"blockBuildWhenUpstreamBuilding"`
	Triggers                         string           `xml:"triggers"`
	ConcurrentBuild                  bool             `xml:"concurrentBuild"`
	CustomWorkspace                  string           `xml:"customWorkspace"`
	Builders                         []JenkinsBuilder `xml:"builders"`
	Publishers                       string           `xml:"publishers"`
	BuildWrappers                    string           `xml:"buildWrappers"`
}

type JenkinsSCM struct {
	Class                             string `xml:"class,attr"`
	Plugin                            string `xml:"plugin,attr"`
	ConfigVersion                     int    `xml:"configVersion"`
	GitRepo                           string `xml:"userRemoteConfigs>hudson.plugins.git.UserRemoteConfig>url"`
	GitBranch                         string `xml:"branches>hudson.plugins.git.BranchSpec>name"`
	DoGenerateSubmoduleConfigurations bool   `xml:"doGenerateSubmoduleConfigurations"`
	SubmodelCfg                       string `xml:"submoduleCfg,omitempty"`
	extensions                        string `xml:"extensions"`
}

type JenkinsBuilder struct {
	Command string `xml:"hudson.tasks.Shell>command"`
}

type JenkinsBuild struct {
	Id                string `json:"id,omitempty"`
	KeepLog           bool   `json:"keepLog,omitempty"`
	Number            int
	QueueId           int
	Result            string
	TimeStamp         int64
	BuiltOn           string
	ChangeSet         ChangeSet
	Duration          int
	EstimatedDuration int
	Building          bool
}

type ChangeSet struct {
	Kind  string
	Items []interface{}
}
