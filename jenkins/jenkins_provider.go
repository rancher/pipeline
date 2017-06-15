package jenkins

import (
	"text/template"

	"bytes"

	"path"

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
	jobName := j.pipeline.Name + "_" + stage.Name
	workspaceBasePath, _ := JenkinsConfig.Get(JenkinsBaseWorkspacePath)
	workspaceName := path.Join(workspaceBasePath, jobName+"_"+j.pipeline.VersionSequence)
	filePath, _ := JenkinsConfig.Get(JenkinsTemlpateFolder)
	tpl, err := template.New("build").ParseFiles(filePath)
	if err != nil {
		return err
	}
	strbuff := new(bytes.Buffer)
	buildData := BuildStruct{
		Repository: j.pipeline.Repository,
		Branch:     j.pipeline.Branch,
		Workspace:  workspaceName,
		Command:    stage.Steps[0].Command,
	}
	if err := tpl.Execute(strbuff, buildData); err != nil {
		return err
	}
	return nil
}

func commandBuilder(workspaceName string, step *pipeline.Step) string {
	stringBuilder := new(bytes.Buffer)
	switch step.Type {
	case pipeline.StepTypeTask:
		volumeInfo := "-v " + workspaceName + ":/workspace/"

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
	return ""
}
