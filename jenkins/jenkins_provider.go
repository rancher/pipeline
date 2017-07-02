package jenkins

import (
	"encoding/xml"
	"fmt"
	"time"

	"bytes"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/restfulserver"
	"github.com/sluu99/uuid"
)

type BuildStruct struct {
	Repository string
	Branch     string
	Workspace  string
	Command    string
}

type JenkinsProvider struct {
	pipeline *pipeline.Pipeline
}

func (j *JenkinsProvider) Init(pipeline *pipeline.Pipeline) error {
	j.pipeline = pipeline
	println("get in provider")
	return nil
}

func (j *JenkinsProvider) RunPipeline(p *pipeline.Pipeline) error {
	j.Init(p)

	//init and create  activity
	activity := pipeline.Activity{
		Id:       uuid.Rand().Hex(),
		Pipeline: *p,
		Status:   "running",
		StartTS:  time.Now().Unix(),
	}
	_, err := restfulserver.CreateActivity(activity)
	if err != nil {
		return err
	}
	//provider run
	if len(p.Stages) > 0 {
		logrus.Info("building")
		if err := j.RunBuild(p.Stages[0], activity.Id); err != nil {
			logrus.Error(errors.Wrap(err, "build stage fail"))
			return err
		}
	}
	logrus.Info("running other test")
	for i := 1; i < len(p.Stages); i++ {
		if err := j.RunStage(p.Stages[i]); err != nil {
			logrus.Error(errors.Wrapf(err, "stage <%s> fail", p.Stages[i].Name))
			return err
		}
	}
	return nil
}

func (j *JenkinsProvider) RunStage(stage *pipeline.Stage) error {
	return nil
}

func (j *JenkinsProvider) RunBuild(stage *pipeline.Stage, activityId string) error {
	logrus.Info("begin jenkins building")
	jobName := j.pipeline.Name + "_" + stage.Name + "_" + activityId

	conf, err := j.generateJenkinsProject(stage)
	if err != nil {
		return err
	}

	if err := CreateJob(jobName, conf); err != nil {
		return err
	}
	if _, err := BuildJob(jobName, map[string]string{}); err != nil {
		return err
	}

	return nil
}

//get build infos and update activity
func (j *JenkinsProvider) SyncActivityInfo(activity *pipeline.Activity) {

}

func (j *JenkinsProvider) generateJenkinsProject(stage *pipeline.Stage) ([]byte, error) {
	logrus.Info("generating jenkins project config")
	jobName := j.pipeline.Name + "_" + stage.Name
	workspaceName := path.Join("${JENKINS_HOME}", "workspace", jobName+"_"+j.pipeline.VersionSequence)

	commandBuilders := []JenkinsBuilder{}
	for _, step := range stage.Steps {
		commandBuilders = append(commandBuilders, JenkinsBuilder{Command: commandBuilder(step)})
	}

	v := &JenkinsProject{
		Scm: JenkinsSCM{
			Class:         "hudson.plugins.git.GitSCM",
			Plugin:        "git@3.3.1",
			ConfigVersion: 2,
			GitRepo:       j.pipeline.Repository,
			GitBranch:     j.pipeline.Branch,
		},
		CanRoam:  true,
		Disabled: false,
		BlockBuildWhenDownstreamBuilding: false,
		BlockBuildWhenUpstreamBuilding:   false,
		CustomWorkspace:                  workspaceName,
		Builders:                         commandBuilders,
	}
	output, err := xml.MarshalIndent(v, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return nil, err
	}
	return output, nil

}

func commandBuilder(step *pipeline.Step) string {
	stringBuilder := new(bytes.Buffer)
	switch step.Type {
	case pipeline.StepTypeTask:
		volumeInfo := "-v ${PWD}:${PWD}"
		stringBuilder.WriteString("docker run --rm")
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(volumeInfo)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(step.Image)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(step.Command)
	case pipeline.StepTypeCatalog:
	case pipeline.StepTypeDeploy:
	}
	logrus.Infof("Finish building command for step command is %s", stringBuilder.String())
	return stringBuilder.String()
}
