package jenkins

import (
	"encoding/xml"
	"fmt"
	"testing"
)

func TestModel(t *testing.T) {
	v := &JenkinsProject{
		Scm: JenkinsSCM{
			Class:         "hudson.plugins.git.GitSCM",
			Plugin:        "git@3.3.1",
			ConfigVersion: 2,
			GitRepo:       "https://github.com/rancher/pipeline.git",
			GitBranch:     "master",
		},
		CanRoam:  true,
		Disabled: false,
		BlockBuildWhenDownstreamBuilding: false,
		BlockBuildWhenUpstreamBuilding:   false,
		CustomWorkspace:                  "$JENKINS_HOME/workspace/test",
		Builders:                         JenkinsBuilder{[]JenkinsTaskShell{JenkinsTaskShell{Command: "echo build1"}, JenkinsTaskShell{Command: "echo build2"}}},
		TimeStampWrapper:                 TimestampWrapperPlugin{Plugin: "timestamper@1.8.8"},
	}
	output, err := xml.MarshalIndent(v, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	t.Logf("output is :%v", string(output))
}
