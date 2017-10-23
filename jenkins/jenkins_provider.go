package jenkins

import (
	"encoding/xml"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bytes"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/restfulserver"
	"github.com/rancher/pipeline/util"
	"github.com/sluu99/uuid"
)

type JenkinsProvider struct {
}

func (j *JenkinsProvider) Init(pipeline *pipeline.Pipeline) error {
	return nil
}

func (j *JenkinsProvider) RunPipeline(p *pipeline.Pipeline, triggerType string) (*pipeline.Activity, error) {

	activity, err := ToActivity(p)
	if err != nil {
		return &pipeline.Activity{}, err
	}
	activity.TriggerType = triggerType
	initActivityEnvvars(activity)

	if len(p.Stages) == 0 {
		return &pipeline.Activity{}, errors.New("no stage in pipeline definition to run!")
	}
	for i := 0; i < len(p.Stages); i++ {
		logrus.Debugf("creating stage:%v", p.Stages[i])
		if err := j.CreateStage(activity, i); err != nil {
			logrus.Error(errors.Wrapf(err, "stage <%s> fail", p.Stages[i].Name))
			return &pipeline.Activity{}, err
		}
	}
	logrus.Debugf("running stage:%v", p.Stages[0])
	if err = j.RunStage(activity, 0); err != nil {
		return &pipeline.Activity{}, err
	}

	logrus.Debugf("creating activity:%v", activity)
	_, err = restfulserver.CreateActivity(*activity)
	if err != nil {
		return &pipeline.Activity{}, err
	}

	return activity, nil
}

//RerunActivity runs an existing activity
func (j *JenkinsProvider) RerunActivity(a *pipeline.Activity) error {
	//find an available node to run
	nodeName, err := getNodeNameToRun()
	if err != nil {
		return err
	}
	a.NodeName = nodeName
	//set to original git commit
	err = j.SetSCMCommit(a)
	if err != nil {
		logrus.Errorf("set scm commit fail,%v", err)
	}

	logrus.Infof("rerunpipeline,get nodeName:%v", nodeName)
	a.RunSequence = a.Pipeline.RunCount + 1
	a.StartTS = time.Now().UnixNano() / int64(time.Millisecond)
	initActivityEnvvars(a)
	err = j.RunStage(a, 0)
	return err
}

//CreateStage init jenkins projects settings of the stage, each step forms a jenkins job.
func (j *JenkinsProvider) CreateStage(activity *pipeline.Activity, ordinal int) error {
	logrus.Info("create jenkins job from stage")
	stage := activity.ActivityStages[ordinal]
	for i, _ := range stage.ActivitySteps {
		conf := j.generateStepJenkinsProject(activity, ordinal, i)
		jobName := getJobName(activity, ordinal, i)
		bconf, _ := xml.MarshalIndent(conf, "  ", "    ")
		if err := CreateJob(jobName, bconf); err != nil {
			return err
		}
	}
	return nil
}

//getNodeNameToRun gets a random node name to run
func getNodeNameToRun() (string, error) {
	nodes, err := GetActiveNodesName()
	if err != nil || len(nodes) == 0 {
		return "", errors.Wrapf(err, "fail to find an active node to work")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	index := r.Intn(len(nodes))
	logrus.Debugf("pick %s to work", nodes[index])
	return nodes[index], nil
}

//DeleteFormerBuild delete last build info of a completed activity
func (j *JenkinsProvider) DeleteFormerBuild(activity *pipeline.Activity) error {
	if activity.Status == pipeline.ActivityBuilding || activity.Status == pipeline.ActivityWaiting {
		return errors.New("cannot delete lastbuild of running activity!")
	}
	for stageOrdinal, stage := range activity.ActivityStages {
		for stepOrdinal, step := range stage.ActivitySteps {
			jobName := getJobName(activity, stageOrdinal, stepOrdinal)
			if step.Status == pipeline.ActivityStepSuccess || step.Status == pipeline.ActivityStepFail {
				logrus.Infof("deleting:%v", jobName)
				if err := DeleteBuild(jobName); err != nil {
					return err
				}
			}
		}
	}
	return nil

}

//SetSCMCommit update jenkins job SCM to use commit id in activity
func (j *JenkinsProvider) SetSCMCommit(activity *pipeline.Activity) error {
	if activity.CommitInfo == "" {
		return errors.New("no commit info in activity")
	}
	conf := j.generateStepJenkinsProject(activity, 0, 0)
	conf.Scm.GitBranch = activity.CommitInfo

	jobName := getJobName(activity, 0, 0)
	bconf, _ := xml.MarshalIndent(conf, "  ", "    ")
	logrus.Debugf("conf:\n%v", string(bconf))
	logrus.Debugf("trying to set commit")
	if err := UpdateJob(jobName, bconf); err != nil {
		logrus.Errorf("updatejob error:%v", err)
		return err
	}
	return nil
}

func EvaluateConditions(activity *pipeline.Activity, condition *pipeline.PipelineConditions) (bool, error) {
	if condition == nil || (len(condition.All) == 0 && len(condition.Any) == 0) {
		return false, fmt.Errorf("Nil condition")
	}
	if len(condition.All) > 0 {
		for _, c := range condition.All {
			resCond, err := EvaluateCondition(activity, c)
			if err != nil {
				return false, err
			}
			if !resCond {
				return false, nil
			}
		}
		return true, nil
	}

	for _, c := range condition.Any {
		resCond, err := EvaluateCondition(activity, c)
		if err != nil {
			return false, err
		}
		if resCond {
			return true, nil
		}
	}
	return false, nil
}

//valid format:     xxx=xxx; xxx!=xxx
func EvaluateCondition(activity *pipeline.Activity, condition string) (bool, error) {
	m := util.GetParams(`(?P<Key>.*?)!=(?P<Value>.*)`, condition)
	if m["Key"] != "" && m["Value"] != "" {
		envVal := activity.EnvVars[m["Key"]]
		if envVal != m["Value"] {
			return true, nil
		}
		return false, nil
	}

	m = util.GetParams(`(?P<Key>.*?)=(?P<Value>.*)`, condition)
	if m["Key"] != "" && m["Value"] != "" {
		envVal := activity.EnvVars[m["Key"]]
		if envVal == m["Value"] {
			return true, nil
		}
		return false, nil
	}
	return false, fmt.Errorf("cannot parse condition:%s", condition)
}

func (j *JenkinsProvider) RunStage(activity *pipeline.Activity, ordinal int) error {
	if len(activity.ActivityStages) <= ordinal {
		return fmt.Errorf("error run stage,stage index out of range")
	}
	stage := activity.Pipeline.Stages[ordinal]
	logrus.Infof("run stage:%s", stage.Name)
	logrus.Debugf("paras:%v,%v,%v,%v", activity.Pipeline, activity, len(activity.Pipeline.Stages), ordinal)
	condFlag := true
	curTime := time.Now().UnixNano() / int64(time.Millisecond)
	var err error
	if pipeline.HasStageCondition(stage) {
		condFlag, err = EvaluateConditions(activity, stage.Conditions)
		if err != nil {
			logrus.Errorf("Evaluate condition '%v' got error:%v", stage.Conditions, err)
			return err
		}
	}
	if !condFlag {
		activity.ActivityStages[ordinal].Status = pipeline.ActivityStageSkip
		if ordinal == len(activity.ActivityStages)-1 {
			//skip last stage and success activity
			activity.Status = pipeline.ActivitySuccess
			activity.StopTS = curTime
			j.OnActivityCompelte(activity)
		} else {
			//skip the stage then run next one.
			err = j.RunStage(activity, ordinal+1)
		}
		return err
	}

	activity.ActivityStages[ordinal].StartTS = curTime
	//Trigger all step jobs in the stage.
	if stage.Parallel {
		for i := 0; i < len(stage.Steps); i++ {
			if err := j.RunStep(activity, ordinal, i); err != nil {
				logrus.Errorf("run step error:%v", err)
				return err
			}
		}
	} else {
		//Trigger first to run sequentially
		if err := j.RunStep(activity, ordinal, 0); err != nil {
			logrus.Errorf("run step error:%v", err)
			return err
		}
	}

	return nil
}

func (j *JenkinsProvider) RunStep(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) error {
	if len(activity.ActivityStages) <= stageOrdinal ||
		len(activity.ActivityStages[stageOrdinal].ActivitySteps) <= stepOrdinal ||
		stageOrdinal < 0 || stepOrdinal < 0 {
		return fmt.Errorf("error run stage,stage index out of range")
	}
	stage := activity.Pipeline.Stages[stageOrdinal]
	step := stage.Steps[stepOrdinal]
	condFlag := true
	var err error
	if pipeline.HasStepCondition(step) {
		condFlag, err = EvaluateConditions(activity, step.Conditions)
		if err != nil {
			logrus.Errorf("Evaluate condition '%v' got error:%v", step.Conditions, err)
			return err
		}
	}
	if !condFlag {
		activity.ActivityStages[stageOrdinal].ActivitySteps[stepOrdinal].Status = pipeline.ActivityStepSkip
		actiStage := activity.ActivityStages[stageOrdinal]
		curTime := time.Now().UnixNano() / int64(time.Millisecond)
		if restfulserver.IsStageSuccess(actiStage) {
			//if skipped and stage success
			actiStage.Status = pipeline.ActivityStageSuccess
			actiStage.Duration = curTime - actiStage.StartTS
			if stageOrdinal == len(activity.ActivityStages)-1 {
				//last stage success and success activity
				activity.Status = pipeline.ActivitySuccess
				activity.StopTS = curTime
				j.OnActivityCompelte(activity)
			} else {
				//success the stage then run next one.
				err = j.RunStage(activity, stageOrdinal+1)
			}
		} else if !stage.Parallel {
			//sequential, skipped current step then run next step
			err = j.RunStep(activity, stageOrdinal, stepOrdinal+1)
		}
		return err
	}
	logrus.Debugf("Run step:%s,%d,%d", activity.Pipeline.Name, stageOrdinal, stepOrdinal)
	jobName := getJobName(activity, stageOrdinal, stepOrdinal)
	if _, err := BuildJob(jobName, map[string]string{}); err != nil {
		logrus.Errorf("run %s error:%v", jobName, err)
		return err
	}

	return nil
}

func (j *JenkinsProvider) generateStepJenkinsProject(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) *JenkinsProject {
	logrus.Info("generating jenkins project config")
	activityId := activity.Id
	workspaceName := path.Join("${JENKINS_HOME}", "workspace", activityId)
	stage := activity.Pipeline.Stages[stageOrdinal]
	step := stage.Steps[stepOrdinal]

	step.Services = pipeline.GetServices(activity, stageOrdinal, stepOrdinal)
	taskShells := []JenkinsTaskShell{}
	taskShells = append(taskShells, JenkinsTaskShell{Command: commandBuilder(activity, step)})
	commandBuilders := JenkinsBuilder{TaskShells: taskShells}

	scm := JenkinsSCM{Class: "hudson.scm.NullSCM"}

	postBuildSctipt := stepFinishScript
	if step.Type == pipeline.StepTypeSCM {
		scm = JenkinsSCM{
			Class:         "hudson.plugins.git.GitSCM",
			Plugin:        "git@3.3.1",
			ConfigVersion: 2,
			GitRepo:       step.Repository,
			GitBranch:     step.Branch,
		}
		postBuildSctipt = stepSCMFinishScript
	}
	preSCMStep := PreSCMBuildStepsWrapper{
		Plugin:      "preSCMbuildstep@0.3",
		FailOnError: false,
		Command:     fmt.Sprintf(stepStartScript, url.QueryEscape(activityId), stageOrdinal, stepOrdinal),
	}

	v := &JenkinsProject{
		Scm:          scm,
		AssignedNode: activity.NodeName,
		CanRoam:      false,
		Disabled:     false,
		BlockBuildWhenDownstreamBuilding: false,
		BlockBuildWhenUpstreamBuilding:   false,
		CustomWorkspace:                  workspaceName,
		Builders:                         commandBuilders,
		TimeStampWrapper:                 TimestampWrapperPlugin{Plugin: "timestamper@1.8.8"},
		PreSCMBuildStepsWrapper:          preSCMStep,
	}
	//post task to notify pipelineserver
	pbt := PostBuildTask{
		Plugin:             "groovy-postbuild@2.3.1",
		Behavior:           0,
		RunForMatrixParent: false,
		GroovyScript: GroovyScript{
			Plugin:  "script-security@1.30",
			Sandbox: false,
			Script:  fmt.Sprintf(postBuildSctipt, url.QueryEscape(activity.Id), stageOrdinal, stepOrdinal),
		},
	}
	v.Publishers = pbt

	return v

}

func commandBuilder(activity *pipeline.Activity, step *pipeline.Step) string {
	stringBuilder := new(bytes.Buffer)
	stringBuilder.WriteString("set +x \n")
	switch step.Type {
	case pipeline.StepTypeTask:

		envVars := ""
		if len(step.Env) > 0 {
			for _, para := range step.Env {
				envVars += fmt.Sprintf("-e %s ", QuoteShell(para))
			}
		}

		entrypointPara := ""
		argsPara := ""
		svcPara := ""
		if step.ShellScript != "" {
			entrypointPara = "--entrypoint /bin/sh"
			entryFileName := fmt.Sprintf(".r_cicd_entrypoint_%s.sh", util.RandStringRunes(4))
			argsPara = entryFileName

			//write to a sh file,then docker run it
			stringBuilder.WriteString(fmt.Sprintf("cat>%s<<R_CICD_EOF\n", entryFileName))
			stringBuilder.WriteString("set -xe\n")
			cmd := strings.Replace(step.ShellScript, "\\", "\\\\", -1)
			cmd = strings.Replace(cmd, "$", "\\$", -1)
			stringBuilder.WriteString(cmd)
			stringBuilder.WriteString("\nR_CICD_EOF\n")
		} else {
			argsPara = step.Args
		}
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")
		if step.Entrypoint != "" {
			entrypointPara = "--entrypoint " + step.Entrypoint
		}
		//isService
		if step.IsService {
			containerName := activity.Id + step.Alias
			svcPara = "-d --name " + containerName
		}

		//add link service
		linkInfo := ""
		if len(step.Services) > 0 {
			linkInfo += ""
			for _, svc := range step.Services {
				linkInfo += fmt.Sprintf("--link %s:%s ", svc.ContainerName, svc.Name)
			}

		}

		//volumeInfo := "--volumes-from ${HOSTNAME} -w ${PWD}"
		volumeInfo := "-v /var/jenkins_home/workspace:/var/jenkins_home/workspace -w ${PWD}"
		stringBuilder.WriteString("set -xe\n")
		stringBuilder.WriteString("docker run --rm")
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString("--env-file ${PWD}/.r_cicd.env")
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(envVars)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(svcPara)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(volumeInfo)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(entrypointPara)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(linkInfo)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(step.Image)
		stringBuilder.WriteString(" ")
		stringBuilder.WriteString(argsPara)
	case pipeline.StepTypeBuild:
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")
		if step.Dockerfile == "" {
			buildPath := "."
			if step.DockerfilePath != "" {
				buildPath = step.DockerfilePath
			}
			stringBuilder.WriteString("set -xe\n")
			stringBuilder.WriteString("docker build --tag ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(" ")
			stringBuilder.WriteString(buildPath)
			stringBuilder.WriteString(";")
		} else {
			stringBuilder.WriteString("echo " + QuoteShell(step.Dockerfile) + ">.Dockerfile;\n")
			stringBuilder.WriteString("set -xe\n")
			stringBuilder.WriteString("docker build --tag ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(" -f .Dockerfile .;")
		}
		if step.PushFlag {
			stringBuilder.WriteString("\ncihelper pushimage ")
			stringBuilder.WriteString(step.TargetImage)
			stringBuilder.WriteString(";")
		}
	case pipeline.StepTypeSCM:
		//write to a env file that provides the environment variables to use throughout the activity.
		stringBuilder.WriteString("GIT_BRANCH=$(echo $GIT_BRANCH|cut -d / -f 2)\n")
		stringBuilder.WriteString("cat>.r_cicd.env<<R_CICD_EOF\n")
		stringBuilder.WriteString("CICD_GIT_COMMIT=$GIT_COMMIT\n")
		stringBuilder.WriteString("CICD_GIT_BRANCH=$GIT_BRANCH\n")
		stringBuilder.WriteString("CICD_GIT_URL=$GIT_URL\n")
		stringBuilder.WriteString("CICD_PIPELINE_NAME=" + activity.Pipeline.Name + "\n")
		stringBuilder.WriteString("CICD_PIPELINE_ID=" + activity.Pipeline.Id + "\n")
		stringBuilder.WriteString("CICD_TRIGGER_TYPE=" + activity.TriggerType + "\n")
		stringBuilder.WriteString("CICD_NODE_NAME=" + activity.NodeName + "\n")
		stringBuilder.WriteString("CICD_ACTIVITY_ID=" + activity.Id + "\n")
		stringBuilder.WriteString("CICD_ACTIVITY_SEQUENCE=" + strconv.Itoa(activity.RunSequence) + "\n")
		//user defined env vars
		for _, envvar := range activity.Pipeline.Parameters {
			splits := strings.SplitN(envvar, "=", 2)
			if len(splits) != 2 {
				continue
			}
			stringBuilder.WriteString(fmt.Sprintf("%s=%s\n", splits[0], QuoteShell(splits[1])))
		}
		stringBuilder.WriteString("\nR_CICD_EOF\n")

	case pipeline.StepTypeUpgradeService:
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")
		stringBuilder.WriteString("set -xe\n")
		stringBuilder.WriteString("cihelper")
		if step.Endpoint != "" {
			stringBuilder.WriteString(" --envurl ")
			stringBuilder.WriteString(QuoteShell(step.Endpoint))
			stringBuilder.WriteString(" --accesskey ")
			stringBuilder.WriteString(QuoteShell(step.Accesskey))
			stringBuilder.WriteString(" --secretkey ")
			stringBuilder.WriteString(QuoteShell(step.Secretkey))
		} else {
			//read from env var
			stringBuilder.WriteString(" --envurl $CATTLE_URL")
			stringBuilder.WriteString(" --accesskey $CATTLE_ACCESS_KEY")
			stringBuilder.WriteString(" --secretkey $CATTLE_SECRET_KEY")
		}
		stringBuilder.WriteString(" upgrade service ")
		if step.ImageTag != "" {
			stringBuilder.WriteString(" --image ")
			stringBuilder.WriteString(step.ImageTag)
		}
		for k, v := range step.ServiceSelector {
			stringBuilder.WriteString(" --selector ")
			stringBuilder.WriteString(QuoteShell(fmt.Sprintf("%s=%s", k, v)))
		}
		if step.BatchSize > 0 {
			stringBuilder.WriteString(" --batchsize ")
			stringBuilder.WriteString(strconv.Itoa(step.BatchSize))
		}
		if step.Interval != 0 {
			stringBuilder.WriteString(" --interval ")
			stringBuilder.WriteString(strconv.Itoa(step.Interval))
		}
		if step.StartFirst != false {
			stringBuilder.WriteString(" --startfirst")
			stringBuilder.WriteString(" true")
		}
	case pipeline.StepTypeUpgradeStack:
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")
		if step.Endpoint == "" {
			script := fmt.Sprintf(upgradeStackScript, "$CATTLE_URL", "$CATTLE_ACCESS_KEY", "$CATTLE_SECRET_KEY", step.StackName, EscapeShell(activity, step.Compose))
			stringBuilder.WriteString(script)
		} else {
			script := fmt.Sprintf(upgradeStackScript, step.Endpoint, step.Accesskey, step.Secretkey, step.StackName, EscapeShell(activity, step.Compose))
			stringBuilder.WriteString(script)
		}
	case pipeline.StepTypeUpgradeCatalog:
		stringBuilder.WriteString(". ${PWD}/.r_cicd.env\n")

		_, templateName, templateBase, _, _ := templateURLPath(step.ExternalId)

		systemFlag := ""
		if templateBase != "" {
			systemFlag = "--system "
		}
		deployFlag := ""
		if step.DeployFlag {
			deployFlag = "true"
		}

		dockerCompose := ""
		rancherCompose := ""
		readme := ""
		for k, v := range step.Templates {
			if strings.HasPrefix(k, "docker-compose") {
				dockerCompose = v
			} else if strings.HasPrefix(k, "rancher-compose") {
				rancherCompose = v
			} else if k == "README.md" {
				readme = v
			}

		}
		dockerCompose = EscapeShell(activity, dockerCompose)
		rancherCompose = EscapeShell(activity, rancherCompose)
		readme = EscapeShell(activity, readme)
		answers := EscapeShell(activity, step.Answers)

		endpoint := step.Endpoint
		accessKey := step.Accesskey
		secretKey := step.Secretkey

		if endpoint == "" {
			endpoint = "$CATTLE_URL"
			accessKey = "$CATTLE_ACCESS_KEY"
			secretKey = "$CATTLE_SECRET_KEY"
		}

		//TODO Multiple
		gitUserName, _ := restfulserver.GetSingleUserName()
		script := fmt.Sprintf(upgradeCatalogScript, step.Repository, step.Branch, gitUserName, systemFlag, templateName, deployFlag, dockerCompose, rancherCompose, readme, answers, endpoint, accessKey, secretKey, step.StackName)
		stringBuilder.WriteString(script)
	case pipeline.StepTypeDeploy:
	}
	//logrus.Infof("Finish building command for step command is %s", stringBuilder.String())
	return stringBuilder.String()
}

func (j *JenkinsProvider) SyncActivity(activity *pipeline.Activity) error {
	for i, actiStage := range activity.ActivityStages {
		for j, actiStep := range actiStage.ActivitySteps {
			if actiStep.Status == pipeline.ActivityStepFail || actiStep.Status == pipeline.ActivityStepSuccess {
				continue
			}
			jobName := getJobName(activity, i, j)
			jobInfo, err := GetJobInfo(jobName)
			if err != nil {
				//cannot get jobinfo
				logrus.Debugf("got job info:%v,err:%v", jobInfo, err)
				return err
			}

			buildInfo, err := GetBuildInfo(jobName)
			if err != nil {
				if actiStage.NeedApproval && j == 0 {
					//Pending
					actiStage.Status = pipeline.ActivityStagePending
					activity.Status = pipeline.ActivityPending
				}
				break
			}

			if err == nil {
				if buildInfo.Result == "SUCCESS" {
					actiStep.StartTS = buildInfo.Timestamp
					actiStep.Duration = buildInfo.Duration
					actiStep.Status = pipeline.ActivityStepSuccess
					if j == len(actiStage.ActivitySteps)-1 {
						//Stage Success
						actiStage.Status = pipeline.ActivityStageSuccess
						actiStage.Duration = buildInfo.Timestamp + buildInfo.Duration - actiStage.StartTS
					}
				} else if buildInfo.Result == "FAILURE" {
					actiStep.StartTS = buildInfo.Timestamp
					actiStep.Duration = buildInfo.Duration
					actiStep.Status = pipeline.ActivityStepFail
					//Stage Fail
					actiStage.Status = pipeline.ActivityStageFail
					actiStage.Duration = buildInfo.Timestamp + buildInfo.Duration - actiStage.StartTS
					//Activity Fail
					activity.Status = pipeline.ActivityFail
					activity.StopTS = buildInfo.Timestamp + buildInfo.Duration
				} else if buildInfo.Building {
					//Building
					actiStep.StartTS = buildInfo.Timestamp
					actiStep.Status = pipeline.ActivityStepBuilding
					actiStage.Status = pipeline.ActivityStageBuilding
					activity.Status = pipeline.ActivityBuilding
					break
				}

			}

		}
	}
	return nil
}

//SyncActivity gets latest activity info, return true if status if changed
func (j *JenkinsProvider) SyncActivityStale(activity *pipeline.Activity) (bool, error) {
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
		actiStage.Status = pipeline.ActivityStageBuilding
		actiStage.StartTS = buildInfo.Timestamp

		//logrus.Info("get buildinfo result:%v,actiStagestatus:%v", buildInfo.Result, actiStage.Status)
		if err == nil {
			rawOutput, err := GetBuildRawOutput(jobName, 0)
			if err != nil {
				logrus.Infof("got rawOutput:%v,err:%v", rawOutput, err)
			}
			//actiStage.RawOutput = rawOutput
			stepStatusUpdated := parseSteps(actiStage, rawOutput)

			if actiStage.Status == pipeline.ActivityStageFail {
				activity.Status = pipeline.ActivityFail
				updated = true
			} else if actiStage.Status == pipeline.ActivityStageSuccess {
				if i == len(p.Stages)-1 {
					//if all stage success , mark activity as success
					activity.StopTS = buildInfo.Timestamp + buildInfo.Duration
					activity.Status = pipeline.ActivitySuccess
					updated = true
				}
				logrus.Infof("stage success:%v", i)

				if i < len(p.Stages)-1 && activity.Pipeline.Stages[i+1].NeedApprove {
					logrus.Infof("set pending")
					activity.Status = pipeline.ActivityPending
					activity.ActivityStages[i+1].Status = pipeline.ActivityStagePending
					activity.PendingStage = i + 1
				}
			}
			updated = updated || stepStatusUpdated
		}
		if beforeStatus != actiStage.Status {
			updated = true
			logrus.Infof("sync activity %v,updated !", activity.Id)
		}
		logrus.Debugf("after sync,beforestatus and after:%v,%v", beforeStatus, actiStage.Status)
	}

	return updated, nil
}

//OnActivityCompelte helps clean up
func (j *JenkinsProvider) OnActivityCompelte(activity *pipeline.Activity) {
	//clean services in activity
	services := pipeline.GetAllServices(activity)
	containerNames := []string{}
	for _, service := range services {
		containerNames = append(containerNames, service.ContainerName)
	}
	command := "docker rm -f " + strings.Join(containerNames, " ")
	cleanServiceScript := fmt.Sprintf(ScriptSkel, activity.NodeName, strings.Replace(command, "\"", "\\\"", -1))
	logrus.Debugf("cleanservicescript is: %v", cleanServiceScript)
	res, err := ExecScript(cleanServiceScript)
	logrus.Debugf("clean services result:%v,%v", res, err)
	logrus.Infof("activity '%s' complete", activity.Id)
	//clean workspace
	// command = "rm -rf ${System.getenv('JENKINS_HOME')}/workspace/" + activity.Id
	// cleanWorkspaceScript := fmt.Sprintf(ScriptSkel, activity.NodeName, strings.Replace(command, "\"", "\\\"", -1))
	// res, err = ExecScript(cleanWorkspaceScript)
	// logrus.Infof("clean workspace result:%v,%v", res, err)

}
func (j *JenkinsProvider) GetStepLog(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int, paras map[string]interface{}) (string, error) {
	if stageOrdinal < 0 || stageOrdinal >= len(activity.ActivityStages) || stepOrdinal < 0 || stepOrdinal >= len(activity.ActivityStages[stageOrdinal].ActivitySteps) {
		return "", errors.New("ordinal out of range")
	}
	jobName := getJobName(activity, stageOrdinal, stepOrdinal)
	var logText *string
	if val, ok := paras["prevLog"]; ok {
		logText = val.(*string)
	}
	startLine := len(strings.Split(*logText, "\n"))

	rawOutput, err := GetBuildRawOutput(jobName, startLine)
	if err != nil {
		return "", err
	}
	logrus.Debugf("got log:\n%s\n\n%s\n\n%d", *logText, rawOutput, startLine)
	token := "\\n\\w{14}\\s{2}\\[.*?\\].*?\\.sh"
	*logText = *logText + rawOutput
	outputs := regexp.MustCompile(token).Split(*logText, -1)
	if len(outputs) > 1 && stageOrdinal == 0 && stepOrdinal == 0 {
		// SCM
		return outputs[1], nil
	}
	if len(outputs) < 3 {
		//no printed log
		return "", nil
	}
	return outputs[2], nil

}

func getCommit(activity *pipeline.Activity, buildInfo *JenkinsBuildInfo) {
	if activity.CommitInfo != "" {
		return
	}

	logrus.Debugf("try to get commitInfo,action:%v", buildInfo.Actions)
	actions := buildInfo.Actions
	for _, action := range actions {

		logrus.Debugf("lastbuiltrevision:%v", action.LastBuiltRevision.SHA1)
		if action.LastBuiltRevision.SHA1 != "" {
			activity.CommitInfo = action.LastBuiltRevision.SHA1
		}
	}
}

//parse jenkins rawoutput to steps,return true if status updated
func parseSteps(actiStage *pipeline.ActivityStage, rawOutput string) bool {
	token := "\\n\\w{14}\\s{2}\\[.*?\\].*?\\.sh"
	lastStatus := pipeline.ActivityStepBuilding
	var updated bool = false
	if strings.HasSuffix(rawOutput, "  Finished: SUCCESS\n") {
		lastStatus = pipeline.ActivityStepSuccess
		actiStage.Status = pipeline.ActivityStageSuccess
	} else if strings.HasSuffix(rawOutput, "  Finished: FAILURE\n") {
		lastStatus = pipeline.ActivityStepFail
		actiStage.Status = pipeline.ActivityStageFail
	}
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
		parseStepTime(actiStage.ActivitySteps[0], outputs[0], actiStage.StartTS)
		actiStage.Duration = actiStage.ActivitySteps[0].Duration
		return updated
	}
	logrus.Debugf("parsed,len output:%v", len(outputs))
	stageTime := int64(0)
	for i, step := range actiStage.ActivitySteps {
		finishStepNum := len(outputs) - 1
		prevStatus := step.Status
		logrus.Debug("getting step %v", i)
		if i < finishStepNum-1 {
			//passed steps
			step.Status = pipeline.ActivityStepSuccess
			parseStepTime(step, outputs[i+1], actiStage.StartTS)
			stageTime = stageTime + step.Duration
		} else if i == finishStepNum-1 {
			//last run step
			step.Status = lastStatus
			parseStepTime(step, outputs[i+1], actiStage.StartTS)
			stageTime = stageTime + step.Duration
		} else {
			//not run steps
			step.Status = pipeline.ActivityStepWaiting
		}
		if prevStatus != step.Status {
			updated = true
		}
		actiStage.ActivitySteps[i] = step
		logrus.Debugf("now step is %v.", step)
	}
	actiStage.Duration = stageTime
	logrus.Debugf("now actistage is %v.", actiStage)

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

	//compute step duration when done
	step.StartTS = activityStartTS + (durationStart.Nanoseconds() / int64(time.Millisecond))

	if step.Status != pipeline.ActivityStepSuccess && step.Status != pipeline.ActivityStepFail {
		return
	}

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

//ToActivity init an activity from pipeline def
func ToActivity(p *pipeline.Pipeline) (*pipeline.Activity, error) {

	//Find a jenkins slave on which to run
	nodeName, err := getNodeNameToRun()
	if err != nil {
		return &pipeline.Activity{}, err
	}
	activity := &pipeline.Activity{
		Id:          uuid.Rand().Hex(),
		Pipeline:    *p,
		RunSequence: p.RunCount + 1,
		Status:      pipeline.ActivityWaiting,
		StartTS:     time.Now().UnixNano() / int64(time.Millisecond),
		NodeName:    nodeName,
	}
	for _, stage := range p.Stages {
		activity.ActivityStages = append(activity.ActivityStages, ToActivityStage(stage))
	}

	return activity, nil
}

func initActivityEnvvars(activity *pipeline.Activity) {
	p := activity.Pipeline
	vars := map[string]string{}
	vars["CICD_PIPELINE_NAME"] = p.Name
	vars["CICD_PIPELINE_ID"] = p.Id
	vars["CICD_NODE_NAME"] = activity.NodeName
	vars["CICD_ACTIVITY_ID"] = activity.Id
	vars["CICD_ACTIVITY_SEQUENCE"] = strconv.Itoa(activity.RunSequence)
	vars["CICD_GIT_URL"] = p.Stages[0].Steps[0].Repository
	vars["CICD_GIT_BRANCH"] = p.Stages[0].Steps[0].Branch
	vars["CICD_GIT_COMMIT"] = activity.CommitInfo
	vars["CICD_TRIGGER_TYPE"] = activity.TriggerType
	//user defined env vars
	for _, envvar := range activity.Pipeline.Parameters {
		splits := strings.SplitN(envvar, "=", 2)
		if len(splits) != 2 {
			continue
		}
		vars[splits[0]] = splits[1]
	}
	activity.EnvVars = vars
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

func QuoteShell(script string) string {
	//Use double quotes so variable substitution works

	escaped := strings.Replace(script, "\\", "\\\\", -1)
	escaped = strings.Replace(script, "\"", "\\\"", -1)
	escaped = "\"" + escaped + "\""
	return escaped
}

func EscapeShell(activity *pipeline.Activity, script string) string {
	escaped := strings.Replace(script, "\\", "\\\\", -1)
	escaped = strings.Replace(escaped, "$", "\\$", -1)

	for k, v := range activity.EnvVars {
		escaped = strings.Replace(escaped, "\\$"+k+" ", v, -1)
		escaped = strings.Replace(escaped, "\\$"+k+"\n", v, -1)
		escaped = strings.Replace(escaped, "\\${"+k+"}", v, -1)

	}
	return escaped
}

func templateURLPath(path string) (string, string, string, string, bool) {
	pathSplit := strings.Split(path, ":")
	switch len(pathSplit) {
	case 2:
		catalog := pathSplit[0]
		template := pathSplit[1]
		templateSplit := strings.Split(template, "*")
		templateBase := ""
		switch len(templateSplit) {
		case 1:
			template = templateSplit[0]
		case 2:
			templateBase = templateSplit[0]
			template = templateSplit[1]
		default:
			return "", "", "", "", false
		}
		return catalog, template, templateBase, "", true
	case 3:
		catalog := pathSplit[0]
		template := pathSplit[1]
		revisionOrVersion := pathSplit[2]
		templateSplit := strings.Split(template, "*")
		templateBase := ""
		switch len(templateSplit) {
		case 1:
			template = templateSplit[0]
		case 2:
			templateBase = templateSplit[0]
			template = templateSplit[1]
		default:
			return "", "", "", "", false
		}
		return catalog, template, templateBase, revisionOrVersion, true
	default:
		return "", "", "", "", false
	}
}

func getJobName(activity *pipeline.Activity, stageOrdinal int, stepOrdinal int) string {
	stage := activity.ActivityStages[stageOrdinal]
	jobName := strings.Join([]string{activity.Pipeline.Name, activity.Id, stage.Name, strconv.Itoa(stepOrdinal)}, "_")
	return jobName
}

func getStageJobsName(activity *pipeline.Activity, stageOrdinal int) string {
	stage := activity.ActivityStages[stageOrdinal]
	jobsName := []string{}
	for i := 0; i < len(stage.ActivitySteps); i++ {
		stepJobName := getJobName(activity, stageOrdinal, i)
		jobsName = append(jobsName, stepJobName)
	}
	return strings.Join(jobsName, ",")
}
