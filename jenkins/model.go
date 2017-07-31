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
	Scm                              JenkinsSCM             `xml:"scm"`
	CanRoam                          bool                   `xml:"canRoam"`
	Disabled                         bool                   `xml:"disabled"`
	BlockBuildWhenDownstreamBuilding bool                   `xml:"blockBuildWhenDownstreamBuilding"`
	BlockBuildWhenUpstreamBuilding   bool                   `xml:"blockBuildWhenUpstreamBuilding"`
	Triggers                         JenkinsTrigger         `xml:"triggers"`
	ConcurrentBuild                  bool                   `xml:"concurrentBuild"`
	CustomWorkspace                  string                 `xml:"customWorkspace"`
	Builders                         JenkinsBuilder         `xml:"builders,omitempty"`
	Publishers                       string                 `xml:"publishers"`
	BuildWrappers                    TimestampWrapperPlugin `xml:"buildWrappers>hudson.plugins.timestamper.TimestamperBuildWrapper"`
}

type JenkinsSCM struct {
	Class                             string `xml:"class,attr"`
	Plugin                            string `xml:"plugin,attr"`
	ConfigVersion                     int    `xml:"configVersion"`
	GitRepo                           string `xml:"userRemoteConfigs>hudson.plugins.git.UserRemoteConfig>url"`
	GitBranch                         string `xml:"branches>hudson.plugins.git.BranchSpec>name"`
	DoGenerateSubmoduleConfigurations bool   `xml:"doGenerateSubmoduleConfigurations"`
	SubmodelCfg                       string `xml:"submoduleCfg,omitempty"`
	Extensions                        string `xml:"extensions"`
}

type JenkinsTrigger struct {
	BuildTrigger JenkinsBuildTrigger `xml:"jenkins.triggers.ReverseBuildTrigger,omitempty"`
	CronTrigger  JenkinsCronTrigger  `xml:"hudson.triggers.TimerTrigger,omitempty"`
}

type JenkinsBuildTrigger struct {
	Spec                   string `xml:"spec"`
	UpstreamProjects       string `xml:"upstreamProjects"`
	ThresholdName          string `xml:"threshold>name"`
	ThresholdOrdinal       int    `xml:"threshold>ordinal"`
	ThresholdColor         string `xml:"threshold>color"`
	ThresholdCompleteBuild bool   `xml:"threshold>completeBuild"`
}

type JenkinsCronTrigger struct {
	Spec string `xml:"spec"`
}

type TimestampWrapperPlugin struct {
	Plugin string `xml:"plugin,attr"`
}

type JenkinsBuilder struct {
	TaskShells []JenkinsTaskShell `xml:"hudson.tasks.Shell"`
}

type JenkinsTaskShell struct {
	Command string `xml:"command"`
}

type JenkinsBuild struct {
	Id                string    `json:"id,omitempty"`
	KeepLog           bool      `json:"keepLog,omitempty"`
	Number            int       `json:"number,omitempty"`
	QueueId           int       `json:"queueId,omitempty"`
	Result            string    `json:"result,omitempty"`
	TimeStamp         int64     `json:"timestamp,omitempty"`
	BuiltOn           string    `json:"building,omitempty"`
	ChangeSet         ChangeSet `json:"chanSet,omitempty"`
	Duration          int       `json:"duration,omitempty"`
	EstimatedDuration int       `json:"estimatedDuration,omitempty"`
	Building          bool      `json:"building,omitempty"`
}

type ChangeSet struct {
	Kind  string
	Items []interface{}
}

type JenkinsJobInfo struct {
	Class   string `json:"_class"`
	Actions []struct {
		Class string `json:"_class"`
	} `json:"actions"`
	Buildable bool `json:"buildable"`
	Builds    []struct {
		Class  string `json:"_class"`
		Number int64  `json:"number"`
		URL    string `json:"url"`
	} `json:"builds"`
	Color              string      `json:"color"`
	ConcurrentBuild    bool        `json:"concurrentBuild"`
	Description        string      `json:"description"`
	DisplayName        string      `json:"displayName"`
	DisplayNameOrNull  interface{} `json:"displayNameOrNull"`
	DownstreamProjects []struct {
		Class string `json:"_class"`
		Color string `json:"color"`
		Name  string `json:"name"`
		URL   string `json:"url"`
	} `json:"downstreamProjects"`
	FirstBuild struct {
		Class  string `json:"_class"`
		Number int64  `json:"number"`
		URL    string `json:"url"`
	} `json:"firstBuild"`
	FullDisplayName string `json:"fullDisplayName"`
	FullName        string `json:"fullName"`
	HealthReport    []struct {
		Description   string `json:"description"`
		IconClassName string `json:"iconClassName"`
		IconURL       string `json:"iconUrl"`
		Score         int64  `json:"score"`
	} `json:"healthReport"`
	InQueue          bool `json:"inQueue"`
	KeepDependencies bool `json:"keepDependencies"`
	LastBuild        struct {
		Class  string `json:"_class"`
		Number int64  `json:"number"`
		URL    string `json:"url"`
	} `json:"lastBuild"`
	LastCompletedBuild struct {
		Class  string `json:"_class"`
		Number int64  `json:"number"`
		URL    string `json:"url"`
	} `json:"lastCompletedBuild"`
	LastFailedBuild struct {
		Class  string `json:"_class"`
		Number int64  `json:"number"`
		URL    string `json:"url"`
	} `json:"lastFailedBuild"`
	LastStableBuild       interface{} `json:"lastStableBuild"`
	LastSuccessfulBuild   interface{} `json:"lastSuccessfulBuild"`
	LastUnstableBuild     interface{} `json:"lastUnstableBuild"`
	LastUnsuccessfulBuild struct {
		Class  string `json:"_class"`
		Number int64  `json:"number"`
		URL    string `json:"url"`
	} `json:"lastUnsuccessfulBuild"`
	Name            string        `json:"name"`
	NextBuildNumber int64         `json:"nextBuildNumber"`
	Property        []interface{} `json:"property"`
	QueueItem       interface{}   `json:"queueItem"`
	Scm             struct {
		Class string `json:"_class"`
	} `json:"scm"`
	UpstreamProjects []interface{} `json:"upstreamProjects"`
	URL              string        `json:"url"`
}

type JenkinsBuildInfo struct {
	Class   string `json:"_class"`
	Actions []struct {
		Class              string `json:"_class"`
		BuildsByBranchName struct {
			Origin_master struct {
				Class       string      `json:"_class"`
				BuildNumber int64       `json:"buildNumber"`
				BuildResult interface{} `json:"buildResult"`
				Marked      struct {
					SHA1   string `json:"SHA1"`
					Branch []struct {
						SHA1 string `json:"SHA1"`
						Name string `json:"name"`
					} `json:"branch"`
				} `json:"marked"`
				Revision struct {
					SHA1   string `json:"SHA1"`
					Branch []struct {
						SHA1 string `json:"SHA1"`
						Name string `json:"name"`
					} `json:"branch"`
				} `json:"revision"`
			} `json:"origin/master"`
		} `json:"buildsByBranchName"`
		Causes []struct {
			Class            string `json:"_class"`
			ShortDescription string `json:"shortDescription"`
			UserID           string `json:"userId"`
			UserName         string `json:"userName"`
		} `json:"causes"`
		LastBuiltRevision struct {
			SHA1   string `json:"SHA1"`
			Branch []struct {
				SHA1 string `json:"SHA1"`
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"lastBuiltRevision"`
		RemoteUrls []string `json:"remoteUrls"`
		ScmName    string   `json:"scmName"`
	} `json:"actions"`
	Artifacts []interface{} `json:"artifacts"`
	Building  bool          `json:"building"`
	BuiltOn   string        `json:"builtOn"`
	ChangeSet struct {
		Class string        `json:"_class"`
		Items []interface{} `json:"items"`
		Kind  string        `json:"kind"`
	} `json:"changeSet"`
	Description       interface{} `json:"description"`
	DisplayName       string      `json:"displayName"`
	Duration          int64       `json:"duration"`
	EstimatedDuration int64       `json:"estimatedDuration"`
	Executor          interface{} `json:"executor"`
	FullDisplayName   string      `json:"fullDisplayName"`
	ID                string      `json:"id"`
	KeepLog           bool        `json:"keepLog"`
	Number            int64       `json:"number"`
	QueueID           int64       `json:"queueId"`
	Result            string      `json:"result"`
	Timestamp         int64       `json:"timestamp"`
	URL               string      `json:"url"`
}
