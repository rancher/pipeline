package jenkins

import (
	"text/template"

	"bytes"

	"path"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/pipeline/pipeline"
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

func (j *JenkinsProvider) RunStage(stage *pipeline.Stage) error {
	return nil
}

func (j *JenkinsProvider) RunBuild(stage *pipeline.Stage) error {
	logrus.Info("begin jenkins building")
	jobName := j.pipeline.Name + "_" + stage.Name
	workspaceName := path.Join("${JENKINS_HOME}", "workspace", jobName+"_"+j.pipeline.VersionSequence)
	filePath, _ := JenkinsConfig.Get(JenkinsTemlpateFolder)
	tpl := template.Must(template.New(BuildJobStageConfigFile).ParseFiles(path.Join(filePath, BuildJobStageConfigFile)))
	strbuff := new(bytes.Buffer)
	buildData := BuildStruct{
		Repository: j.pipeline.Repository,
		Branch:     j.pipeline.Branch,
		Workspace:  workspaceName,
		Command:    commandBuilder(stage.Steps[0]),
	}
	if err := tpl.Execute(strbuff, buildData); err != nil {
		return err
	}
	if err := CreateJob(jobName, strbuff.Bytes()); err != nil {
		return err
	}
	if _, err := BuildJob(jobName, map[string]string{}); err != nil {
		return err
	}
	return nil
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
