package pipeline

import (
	"github.com/rancher/go-rancher/client"
)

const StepTypeTask = "task"
const StepTypeDeploy = "deploy"
const StepTypeSCM = "scm"
const StepTypeBuild = "build"
const StepTypeService = "service"
const StepTypeUpgradeService = "upgradeService"
const StepTypeUpgradeStack = "upgradeStack"
const StepTypeUpgradeCatalog = "upgradeCatalog"
const TriggerTypeCron = "cron"
const TriggerTypeManual = "manual"
const TriggerTypeWebhook = "webhook"
const (
	ActivityStepWaiting  = "Waiting"
	ActivityStepBuilding = "Building"
	ActivityStepSuccess  = "Success"
	ActivityStepFail     = "Fail"
	ActivityStepSkip     = "Skipped"

	ActivityStageWaiting  = "Waiting"
	ActivityStagePending  = "Pending"
	ActivityStageBuilding = "Building"
	ActivityStageSuccess  = "Success"
	ActivityStageFail     = "Fail"
	ActivityStageDenied   = "Denied"
	ActivityStageSkip     = "Skipped"

	ActivityWaiting  = "Waiting"
	ActivityPending  = "Pending"
	ActivityBuilding = "Building"
	ActivitySuccess  = "Success"
	ActivityFail     = "Fail"
	ActivityDenied   = "Denied"
)

var PreservedEnvs = [...]string{"CICD_GIT_COMMIT", "CICD_GIT_BRANCH",
	"CICD_GIT_URL", "CICD_PIPELINE_NAME", "CICD_PIPELINE_ID",
	"CICD_TRIGGER_TYPE", "CICD_NODE_NAME", "CICD_ACTIVITY_ID",
	"CICD_ACTIVITY_SEQUENCE",
	// "CICD_GIT_PREVIOUS_COMMIT", "CICD_GIT_PREVIOUS_SUCCESSFUL_COMMIT",
	// "CICD_GIT_LOCAL_BRANCH", "CICD_GIT_COMMITTER_NAME",
	// "CICD_GIT_AUTHOR_NAME", "CICD_GIT_COMMITTER_EMAIL", "CICD_GIT_AUTHOR_EMAIL", "CICD_SVN_REVISION",
	// "CICD_SVN_URL",
}

type GithubAccount struct {
	ID          int    `json:"id,omitempty"`
	Login       string `json:"login,omitempty"`
	Name        string `json:"name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	HTMLURL     string `json:"html_url,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
}

type PipelineSetting struct {
	client.Resource
	IsAuth             bool            `json:"isAuth,omitempty" yaml:"isAuth,omitempty"`
	GithubHostName     string          `json:"githubHostName,omitempty" yaml:"githubHostName,omitempty"`
	GithubSchema       string          `json:"githubSchema,omitempty" yaml:"githubSchema,omitempty"`
	GithubHomePage     string          `json:"githubHomepage,omitempty" yaml:"githubHomepage,omitempty"`
	GithubClientID     string          `json:"githubClientID,omitempty" yaml:"githubClientID,omitempty"`
	GithubClientSecret string          `json:"githubClientSecret,omitempty" yaml:"githubClientSecret,omitempty"`
	GithubRedirectURL  string          `json:"githubRedirectURL,omitempty" yaml:"githubRedirectURL,omitempty"`
	GithubAccounts     []GithubAccount `json:"githubAccounts,omitempty" yaml:"githubAccounts,omitempty"`
}

type Pipeline struct {
	client.Resource
	PipelineContent
}

type PipelineContent struct {
	Name            string `json:"name,omitempty" yaml:"name,omitempty"`
	IsActivate      bool   `json:"isActivate" yaml:"isActivate"`
	VersionSequence string `json:"-" yaml:"-"`
	RunCount        int    `json:"runCount" yaml:"runCount,omitempty"`
	LastRunId       string `json:"lastRunId,omitempty" yaml:"lastRunId,omitempty"`
	LastRunStatus   string `json:"lastRunStatus,omitempty" yaml:"lastRunStatus,omitempty"`
	LastRunTime     int64  `json:"lastRunTime,omitempty" yaml:"lastRunTime,omitempty"`
	NextRunTime     int64  `json:"nextRunTime,omitempty" yaml:"nextRunTime,omitempty"`
	CommitInfo      string `json:"commitInfo,omitempty" yaml:"commitInfo,omitempty"`
	Repository      string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch          string `json:"branch,omitempty" yaml:"branch,omitempty"`
	TargetImage     string `json:"targetImage,omitempty" yaml:"target-image,omitempty"`
	File            string `json:"file,omitempty" yaml:"file,omitempty"`
	WebHookId       int    `json:"webhookId,omitempty" yaml:"webhookId,omitempty"`
	WebHookToken    string `json:"webhookToken,omitempty" yaml:"webhookToken,omitempty"`
	//for import
	Templates map[string]string `json:"templates,omitempty" yaml:"templates,omitempty"`
	//trigger
	CronTrigger *CronTrigger `json:"cronTrigger,omitempty" yaml:"cronTrigger,omitempty"`
	Stages      []*Stage     `json:"stages,omitempty" yaml:"stages,omitempty"`
}

type CronTrigger struct {
	TriggerOnUpdate bool   `json:"triggerOnUpdate" yaml:"triggerOnUpdate,omitempty"`
	Spec            string `json:"spec,omitempty" yaml:"spec,omitempty"`
	Timezone        string `json:"timezone,omitempty" yaml:"timezone,omitempty"`
}

type Stage struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	NeedApprove bool   `json:"needApprove" yaml:"needApprove,omitempty"`
	Parallel    bool   `json:"parallel" yaml:"parallel,omitempty"`
	//Condition   string             `json:"condition,omitempty" yaml:"condition,omitempty"`
	Conditions *PipelineConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Approvers  []string            `json:"approvers,omitempty" yaml:"approvers,omitempty"`
	Steps      []*Step             `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type Step struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	//Condition  string             `json:"condition,omitempty" yaml:"condition,omitempty"`
	Conditions *PipelineConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	//---SCM step
	SCMType    string `json:"scmType,omitempty" yaml:"scmType,omitempty"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch     string `json:"branch,omitempty" yaml:"branch,omitempty"`
	Webhook    bool   `json:"webhook" yaml:"webhook,omitempty"`
	//---Build step
	Dockerfile     string `json:"dockerFileContent,omitempty" yaml:"dockerFileContent,omitempty"`
	DockerfilePath string `json:"dockerFilePath,omittempty" yaml:"dockerFilePath,omitempty"`
	TargetImage    string `json:"targetImage,omitempty" yaml:"targetImage,omitempty"`
	PushFlag       bool   `json:"push" yaml:"push,omitempty"`

	//---task step
	Image       string       `json:"image,omitempty" yaml:"image,omitempty"`
	IsService   bool         `json:"isService" yaml:"isService,omitempty"`
	Alias       string       `json:"alias,omitempty" yaml:"alias,omitempty"`
	ShellScript string       `json:"shellScript,omitempty" yaml:"shellScript,omitempty"`
	Entrypoint  string       `json:"entrypoint,omitempty" yaml:"enrtypoint,omitempty"`
	Args        string       `json:"args,omitempty" yaml:"args,omitempty"`
	Env         []string     `json:"env,omitempty" yaml:"env,omitempty"`
	Services    []*CIService `json:"services,omitempty" yaml:"services,omitempty"`

	//---upgradeService step
	ImageTag        string            `json:"imageTag,omitempty" yaml:"imageTag,omitempty"`
	ServiceSelector map[string]string `json:"serviceSelector,omitempty" yaml:"serviceSelector,omitempty"`
	BatchSize       int               `json:"batchSize,omitempty" yaml:"batchSize,omitempty"`
	Interval        int               `json:"interval,omitempty" yaml:"interval,omitempty"`
	StartFirst      bool              `json:"startFirst" yaml:"startFirst,omitempty"`
	Endpoint        string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Accesskey       string            `json:"accesskey,omitempty" yaml:"accesskey,omitempty"`
	Secretkey       string            `json:"secretkey,omitempty" yaml:"secretkey,omitempty"`

	//---upgradeStack step
	//Endpoint,Accesskey,Secretkey
	StackName string `json:"stackName,omitempty" yaml:"stackName,omitempty"`
	Compose   string `json:"compose,omitempty" yaml:"compose,omitempty"`

	//---upgradeCatalog step
	//Endpoint,Accesskey,Secretkey,StackName,
	ExternalId string            `json:"externalId,omitempty" yaml:"externalId,omitempty"`
	DeployFlag bool              `json:"deploy" yaml:"deploy,omitempty"`
	Templates  map[string]string `json:"templates,omitempty" yaml:"templates,omitempty"`
	Answers    string            `json:"answerString,omitempty" yaml:"answerString,omitempty"`
}

type PipelineProvider interface {
	RunPipeline(*Pipeline, string) (*Activity, error)
	RerunActivity(*Activity) error
	RunStage(*Activity, int) error
	RunStep(*Activity, int, int) error
	SyncActivity(*Activity) error
	GetStepLog(*Activity, int, int, map[string]interface{}) (string, error)
	DeleteFormerBuild(*Activity) error
	OnActivityCompelte(*Activity)
}

type PipelineConditions struct {
	All []string `json:"all,omitempty" yaml:"all,omitempty"`
	Any []string `json:"any,omitempty" yaml:"any,omitempty"`
}

type Activity struct {
	client.Resource
	Id              string            `json:"id,omitempty"`
	Pipeline        Pipeline          `json:"pipelineSource,omitempty"`
	PipelineName    string            `json:"pipelineName,omitempty"`
	PipelineVersion string            `json:"pipelineVersion,omitempty"`
	RunSequence     int               `json:"runSequence,omitempty"`
	CommitInfo      string            `json:"commitInfo,omitempty"`
	Status          string            `json:"status,omitempty"`
	FailMessage     string            `json:"failMessage,omitempty"`
	PendingStage    int               `json:"pendingStage,omitempty"`
	StartTS         int64             `json:"start_ts,omitempty"`
	StopTS          int64             `json:"stop_ts,omitempty"`
	NodeName        string            `json:"nodename,omitempty"`
	ActivityStages  []*ActivityStage  `json:"activity_stages,omitempty"`
	EnvVars         map[string]string `json:"envVars,omitempty"`
}

type ActivityStage struct {
	ActivityId    string          `json:"activity_id,omitempty"`
	Name          string          `json:"name,omitempty"`
	NeedApproval  bool            `json:"need_approval,omitempty"`
	Approvers     []string        `json:"approvers,omitempty"`
	ActivitySteps []*ActivityStep `json:"activity_steps,omitempty"`
	StartTS       int64           `json:"start_ts,omitempty"`
	Duration      int64           `json:"duration,omitempty"`
	Status        string          `json:"status,omitempty"`
	RawOutput     string          `json:"rawOutput,omitempty"`
}

type ActivityStep struct {
	Name     string `json:"name,omitempty"`
	Message  string `json:"message,omitempty"`
	Status   string `json:"status,omitempty"`
	StartTS  int64  `json:"start_ts,omitempty"`
	Duration int64  `json:"duration,omitempty"`
}

type CIService struct {
	ContainerName string `json:"containerName,omitempty"`
	Name          string `json:"name,omitempty"`
	Image         string `json:"image,omitempty"`
	Entrypoint    string `json:"entrypoint,omitempty"`
	Command       string `json:"command,omitempty"`
}
