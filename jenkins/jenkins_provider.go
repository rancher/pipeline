package jenkins

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
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

func (j *JenkinsProvider) RunPipeline(p *pipeline.Pipeline) (*pipeline.Activity, error) {
	j.Init(p)

	//init and create  activity
	activity := pipeline.Activity{
		Id:          uuid.Rand().Hex(),
		Pipeline:    *p,
		RunSequence: p.RunCount + 1,
		Status:      pipeline.ActivityWaiting,
		StartTS:     time.Now().UnixNano() / int64(time.Millisecond),
	}
	for _, stage := range p.Stages {
		activity.ActivityStages = append(activity.ActivityStages, ToActivityStage(stage))
	}
	//logrus.Infof("creating activity:%v", activity)
	_, err := restfulserver.CreateActivity(activity)
	if err != nil {
		return &pipeline.Activity{}, err
	}
	//provider run
	/*
		if len(p.Stages) > 0 {
			logrus.Info("building")
			if err := j.RunBuild(p.Stages[0], activity.Id); err != nil {
				logrus.Error(errors.Wrap(err, "build stage fail"))
				return err
			}
		}*/
	if len(p.Stages) == 0 {
		return &pipeline.Activity{}, errors.New("no stage in pipeline definition to run!")
	}
	for i := 0; i < len(p.Stages); i++ {
		//logrus.Infof("creating stage:%v", p.Stages[i])
		if err := j.CreateStage(&activity, i); err != nil {
			logrus.Error(errors.Wrapf(err, "stage <%s> fail", p.Stages[i].Name))
			return &pipeline.Activity{}, err
		}
	}
	//logrus.Infof("running stage:%v", p.Stages[0])
	err = j.RunStage(&activity, 0)

	if err != nil {
		return &pipeline.Activity{}, err
	}

	return &activity, nil
}

//CreateStage init jenkins project settings of the stage
func (j *JenkinsProvider) CreateStage(activity *pipeline.Activity, ordinal int) error {
	logrus.Info("create jenkins job from stage")
	stage := activity.Pipeline.Stages[ordinal]
	activityId := activity.Id
	jobName := j.pipeline.Name + "_" + stage.Name + "_" + activityId

	conf, err := j.generateJenkinsProject(activity, ordinal)
	if err != nil {
		return err
	}

	if err := CreateJob(jobName, conf); err != nil {
		return err
	}
	return nil
}

func (j *JenkinsProvider) RunStage(activity *pipeline.Activity, ordinal int) error {
	logrus.Info("begin jenkins stage")
	//logrus.Infof("hi,%v\nhi,%v\nhi,%v\nhi,%v", activity.Pipeline, activity, len(activity.Pipeline.Stages), ordinal)
	stage := activity.Pipeline.Stages[ordinal]
	activityId := activity.Id
	jobName := activity.Pipeline.Name + "_" + stage.Name + "_" + activityId

	if _, err := BuildJob(jobName, map[string]string{}); err != nil {
		return err
	}

	return nil
}

func (j *JenkinsProvider) RunBuild(stage *pipeline.Stage, activityId string) error {
	return nil
}

/*
func (j *JenkinsProvider) RunBuild(stage *pipeline.Stage, activityId string) error {
	logrus.Info("begin jenkins building")
	jobName := j.pipeline.Name + "_" + stage.Name + "_" + activityId

	conf, err := j.generateJenkinsProject(stage, activityId)
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
*/
func (j *JenkinsProvider) generateJenkinsProject(activity *pipeline.Activity, ordinal int) ([]byte, error) {
	logrus.Info("generating jenkins project config")
	stage := activity.Pipeline.Stages[ordinal]
	activityId := activity.Id
	workspaceName := path.Join("${JENKINS_HOME}", "workspace", activityId)

	taskShells := []JenkinsTaskShell{}
	for _, step := range stage.Steps {
		taskShells = append(taskShells, JenkinsTaskShell{Command: commandBuilder(step)})
		//logrus.Infof("get step:%v,commandbuilders:%v", step, commandBuilders)
	}
	commandBuilders := JenkinsBuilder{TaskShells: taskShells}

	scm := JenkinsSCM{Class: "hudson.scm.NullSCM"}
	if len(stage.Steps) == 1 && stage.Steps[0].Type == pipeline.StepTypeSCM {
		scm = JenkinsSCM{
			Class:         "hudson.plugins.git.GitSCM",
			Plugin:        "git@3.3.1",
			ConfigVersion: 2,
			GitRepo:       stage.Steps[0].Repository,
			GitBranch:     stage.Steps[0].Branch,
		}
	}
	v := &JenkinsProject{
		Scm:      scm,
		CanRoam:  true,
		Disabled: false,
		BlockBuildWhenDownstreamBuilding: false,
		BlockBuildWhenUpstreamBuilding:   false,
		CustomWorkspace:                  workspaceName,
		Builders:                         commandBuilders,
		BuildWrappers:                    TimestampWrapperPlugin{Plugin: "timestamper@1.8.8"},
	}
	//logrus.Infof("needapprove:%v,ordinal:%v", stage.NeedApprove, ordinal)
	if !stage.NeedApprove && ordinal > 0 {
		//Add build trigger
		prevJobName := j.pipeline.Name + "_" + j.pipeline.Stages[ordinal-1].Name + "_" + activityId
		v.Triggers = JenkinsTrigger{
			BuildTrigger: JenkinsBuildTrigger{
				UpstreamProjects:       prevJobName,
				ThresholdName:          "SUCCESS",
				ThresholdOrdinal:       0,
				ThresholdColor:         "BLUE",
				ThresholdCompleteBuild: true,
			},
		}
	}

	//logrus.Infof("before xml:%v", v)
	//logrus.Infof("v.Triggers:%v", v.Triggers)
	output, err := xml.MarshalIndent(v, "  ", "    ")
	//logrus.Infof("get xml:%v\n end xml.", string(output))
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
		//write to a sh file,then docker run it
		stringBuilder.WriteString("cat>.r_cicd_entrypoint.sh<<EOF\n")
		cmd := strings.Replace(step.Command, "\\", "\\\\", -1)
		cmd = strings.Replace(cmd, "$", "\\$", -1)
		stringBuilder.WriteString(cmd)
		stringBuilder.WriteString("\nEOF\n")

		volumeInfo := "--volumes-from ${HOSTNAME} -w ${PWD}"
		stringBuilder.WriteString("docker run --rm")
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(volumeInfo)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(step.Image)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString("/bin/sh -xe .r_cicd_entrypoint.sh")
		/*
			cmdAllBuf := new(bytes.Buffer)
			cmdAllBuf.WriteString("/bin/sh -xe .r_cicd_entrypoint.sh")

				cmdAllBuf.WriteString(strings.Replace(step.Command, "\"", "\\\"", -1))

					cmds := strings.Split(step.Command, "\n")
					for _, cmd := range cmds {
						if strings.HasSuffix(cmd, "\\n") {
							cmdAllBuf.WriteString(strings.Replace(cmd, "\"", "\\\"", -1))
							continue
						} else {
							cmdAllBuf.WriteString(strings.Replace(cmd, "\"", "\\\"", -1))
							cmdAllBuf.WriteString(";")
						}
					}
				cmdAllBuf.WriteString("\"")
				stringBuilder.WriteString(cmdAllBuf.String())
		*/
	case pipeline.StepTypeBuild:

		if step.SourceType == "sc" {
			stringBuilder.WriteString("docker build --tag ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(" .;")
		} else if step.SourceType == "file" {
			stringBuilder.WriteString("echo \"")
			stringBuilder.WriteString(strings.Replace(step.Dockerfile, "\"", "\\\"", -1))
			stringBuilder.WriteString("\">.Dockerfile;")
			stringBuilder.WriteString("docker build --tag ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(" -f .Dockerfile .;")
		}
		if step.PushFlag {
			sep := strings.Index(step.TargetImage, "/")
			registry := ""
			if sep != -1 {
				registry = step.TargetImage[:sep]
			}
			stringBuilder.WriteString("docker login --username ")
			stringBuilder.WriteString(step.RegUserName)
			stringBuilder.WriteString(" --password ")
			stringBuilder.WriteString(step.RegPassword)
			stringBuilder.WriteString(" ")
			stringBuilder.WriteString(registry)
			stringBuilder.WriteString(";docker push ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(";")
		}
	case pipeline.StepTypeSCM:
	case pipeline.StepTypeCatalog:
	case pipeline.StepTypeDeploy:
	}
	//logrus.Infof("Finish building command for step command is %s", stringBuilder.String())
	return stringBuilder.String()
}

//SyncActivity gets latest activity info, return true if status if changed
func (j *JenkinsProvider) SyncActivity(activity *pipeline.Activity) (bool, error) {
	p := activity.Pipeline
	var updated bool

	//logrus.Infof("syncing activity:%v", activity.Id)
	//logrus.Infof("activity is:%v", activity)
	for i, actiStage := range activity.ActivityStages {
		jobName := p.Name + "_" + actiStage.Name + "_" + activity.Id
		beforeStatus := actiStage.Status

		if beforeStatus == pipeline.ActivityStageSuccess {
			continue
		}

		jobInfo, err := GetJobInfo(jobName)
		if err != nil {
			//cannot get jobinfo
			logrus.Infof("got job info:%v,err:%v", jobInfo, err)
			return false, err
		}

		/*
			if (jobInfo.LastBuild == JenkinsJobInfo.LastBuild{}) {
				//no build finish
				return nil
			}
		*/
		buildInfo, err := GetBuildInfo(jobName)
		//logrus.Infof("got build info:%v, err:%v", buildInfo, err)
		if err != nil {
			//cannot get build info
			//build not started
			if actiStage.Status == pipeline.ActivityStagePending {
				return updated, nil
			}
			actiStage.Status = pipeline.ActivityStageWaiting
			break
		}
		getCommit(activity, buildInfo)
		//if any buildInfo found,activity in building status
		activity.Status = pipeline.ActivityBuilding
		actiStage.Duration = buildInfo.Duration
		if buildInfo.Result == "" {
			actiStage.Status = pipeline.ActivityStageBuilding
		} else if buildInfo.Result == "FAILURE" {
			actiStage.Status = pipeline.ActivityStageFail
			activity.Status = pipeline.ActivityFail
		} else if buildInfo.Result == "SUCCESS" {
			actiStage.Status = pipeline.ActivityStageSuccess
			if i == len(p.Stages)-1 {
				//if all stage success , mark activity as success
				activity.StopTS = buildInfo.Timestamp + buildInfo.Duration
				activity.Status = pipeline.ActivitySuccess
			}
			logrus.Infof("stage success:%v", i)

			if i < len(p.Stages)-1 && activity.Pipeline.Stages[i+1].NeedApprove {
				logrus.Infof("set pending")
				activity.Status = pipeline.ActivityPending
				activity.ActivityStages[i+1].Status = pipeline.ActivityStagePending
				activity.PendingStage = i + 1
			}
		}
		//logrus.Info("get buildinfo result:%v,actiStagestatus:%v", buildInfo.Result, actiStage.Status)
		if err == nil {
			rawOutput, err := GetBuildRawOutput(jobName)
			if err != nil {
				logrus.Infof("got rawOutput:%v,err:%v", rawOutput, err)
			}
			//actiStage.RawOutput = rawOutput
			stepStatusUpdated := parseSteps(activity, actiStage, rawOutput)

			updated = updated || stepStatusUpdated
		}
		if beforeStatus != actiStage.Status {
			updated = true
			logrus.Infof("sync activity %v,updated !", activity.Id)
		}
		logrus.Infof("after sync,beforestatus and after:%v,%v", beforeStatus, actiStage.Status)
	}

	return updated, nil
}

func (j *JenkinsProvider) GetStepLog(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) (string, error) {
	p := activity.Pipeline
	logrus.Infof("getting step log")
	if stageOrdinal < 0 || stageOrdinal > len(activity.ActivityStages) || stepOrdinal < 0 || stepOrdinal > len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return "", errors.New("ordinal out of range")
	}
	actiStage := activity.ActivityStages[stageOrdinal]
	jobName := p.Name + "_" + actiStage.Name + "_" + activity.Id
	rawOutput, err := GetBuildRawOutput(jobName)
	if err != nil {
		return "", err
	}
	token := "\\n\\w{14}\\s{2}\\[.*?\\].*?\\.sh"
	outputs := regexp.MustCompile(token).Split(rawOutput, -1)
	if len(outputs) > 0 && len(actiStage.ActivitySteps) > 0 && strings.Contains(outputs[0], "  Cloning the remote Git repository\n") {
		// SCM
		return outputs[0], nil
	}
	if len(outputs) < stepOrdinal+2 {
		//no printed log
		return "", nil
	}
	//logrus.Infof("got step log:%v", outputs[stepOrdinal+1])
	return outputs[stepOrdinal+1], nil

}

func getCommit(activity *pipeline.Activity, buildInfo *JenkinsBuildInfo) {
	if activity.CommitInfo != "" {
		return
	}

	//logrus.Infof("try to get commitInfo,action:%v", buildInfo.Actions)
	actions := buildInfo.Actions
	for _, action := range actions {

		//logrus.Infof("lastbuiltrevision:%v", action.LastBuiltRevision.SHA1)
		if action.LastBuiltRevision.SHA1 != "" {
			activity.CommitInfo = action.LastBuiltRevision.SHA1
		}
	}
}

//parse jenkins rawoutput to steps,return true if status updated
func parseSteps(activity *pipeline.Activity, actiStage *pipeline.ActivityStage, rawOutput string) bool {
	token := "\\n\\w{14}\\s{2}\\[.*?\\].*?\\.sh"
	lastStatus := pipeline.ActivityStepBuilding
	var updated bool = false
	//TODO add timestamp
	if strings.HasSuffix(rawOutput, "  Finished: SUCCESS\n") {
		lastStatus = pipeline.ActivityStepSuccess
		actiStage.Status = pipeline.ActivityStageSuccess
	} else if strings.HasSuffix(rawOutput, "  Finished: FAILURE\n") {
		lastStatus = pipeline.ActivityStepFail
		actiStage.Status = pipeline.ActivityStageFail
	}
	//logrus.Infof("raw:%v,\ntoken:%v", rawOutput, token)
	outputs := regexp.MustCompile(token).Split(rawOutput, -1)
	//logrus.Infof("split to %v parts,steps number:%v, parse outputs:%v", len(outputs), len(actiStage.ActivitySteps), outputs)
	if len(outputs) > 0 && len(actiStage.ActivitySteps) > 0 && strings.Contains(outputs[0], "  Cloning the remote Git repository\n") {
		// SCM
		//actiStage.ActivitySteps[0].Message = outputs[0]
		if actiStage.ActivitySteps[0].Status != lastStatus {
			updated = true
			actiStage.ActivitySteps[0].Status = lastStatus
		}
		//get step time for SCM
		parseStepTime(actiStage.ActivitySteps[0], outputs[0], activity.StartTS)
		return updated
	}
	logrus.Infof("parsed,len output:%v", len(outputs))
	for i, step := range actiStage.ActivitySteps {
		finishStepNum := len(outputs) - 1
		prevStatus := step.Status
		logrus.Info("getting step %v", i)
		if i < finishStepNum-1 {
			//passed steps
			//step.Message = outputs[i+1]
			step.Status = pipeline.ActivityStepSuccess
			parseStepTime(step, outputs[i+1], activity.StartTS)
		} else if i == finishStepNum-1 {
			//last run step
			//step.Message = outputs[i+1]
			step.Status = lastStatus
			parseStepTime(step, outputs[i+1], activity.StartTS)
		} else {
			//not run steps
			step.Status = pipeline.ActivityStepWaiting
		}
		if prevStatus != step.Status {
			updated = true
		}
		actiStage.ActivitySteps[i] = step
		logrus.Infof("now step is %v.", step)
	}
	logrus.Infof("now actistage is %v.", actiStage)

	return updated

}

func parseStepTime(step *pipeline.ActivityStep, log string, activityStartTS int64) {
	logrus.Infof("parsesteptime")
	token := "(^|\\n)\\w{14}  "
	r, _ := regexp.Compile(token)
	lines := r.FindAllString(log, -1)
	if len(lines) == 0 {
		return
	}

	start := strings.TrimLeft(lines[0], "\n")
	start = strings.TrimRight(start, " ")
	durationStart, err := time.ParseDuration(start)
	if err != nil {
		logrus.Errorf("parse duration error!%v", err)
		return
	}

	if step.Status != pipeline.ActivityStepSuccess && step.Status != pipeline.ActivityStepFail {
		return
	}
	//compute step duration when done
	step.StartTS = activityStartTS + (durationStart.Nanoseconds() / int64(time.Millisecond))

	end := strings.TrimLeft(lines[len(lines)-1], "\n")
	end = strings.TrimRight(end, " ")
	durationEnd, err := time.ParseDuration(end)
	if err != nil {
		logrus.Errorf("parse duration error!%vparseStepTime", err)
		return
	}
	duration := (durationEnd.Nanoseconds() - durationStart.Nanoseconds()) / int64(time.Millisecond)
	step.Duration = duration
}

func ToActivityStage(stage *pipeline.Stage) *pipeline.ActivityStage {
	actiStage := pipeline.ActivityStage{
		Name:          stage.Name,
		NeedApproval:  stage.NeedApprove,
		Status:        "Waiting",
		ActivitySteps: []*pipeline.ActivityStep{},
	}
	for _, step := range stage.Steps {
		actiStep := &pipeline.ActivityStep{
			Name:   step.Name,
			Status: pipeline.ActivityStepWaiting,
		}
		actiStage.ActivitySteps = append(actiStage.ActivitySteps, actiStep)
	}
	return &actiStage

}
