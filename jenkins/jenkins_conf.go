package jenkins

import (
	"errors"
	"sync"
)

type jenkinsConfig map[string]string

const JenkinsServerAddress = "JenkinsServerAddress"
const JenkinsUser = "JenkinsUser"
const JenkinsToken = "JenkinsToken"
const CreateJobURI = "CreateJobURI"
const UpdateJobURI = "UpdateJobURI"
const ScriptURI = "ScriptURI"
const DeleteBuildURI = "DeleteBuildURI"
const GetCrumbURI = "GetCrumbURI"
const JenkinsCrumbHeader = "JenkinsCrumbHeader"
const JenkinsCrumb = "JenkinsCrumb"
const JenkinsJobBuildURI = "JenkinsJobBuildURI"
const JenkinsJobInfoURI = "JenkinsJobInfoURI"
const JenkinsBuildInfoURI = "JenkinsBuildInfoURI"
const JenkinsBuildLogURI = "JenkinsBuildLogURI"
const JenkinsJobBuildWithParamsURI = "JenkinsJobBuildWithParamsURI"
const JenkinsTemlpateFolder = "JenkinsTemlpateFolder"

//const JenkinsBaseWorkspacePath = "JenkinsBaseWorkspacePath"
const BuildJobStageConfigFile = "build_stage_example.xml"

//Script to execute on specific node
const ScriptSkel = `import hudson.util.RemotingDiagnostics; 
node = "%s"
script = "%s"
cmd = 'def proc = "'+script+'".execute(); proc.waitFor(); println proc.in.text';
for (slave in hudson.model.Hudson.instance.slaves) {
  if(slave.name==node){
	println RemotingDiagnostics.executeGroovy(cmd, slave.getChannel());
  }
}
//on master
if(node == "master"){
	def proc = script.execute(); proc.waitFor(); println proc.in.text
}
`

const GetActiveNodesScript = `for (slave in hudson.model.Hudson.instance.slaves) {
  if (!slave.getComputer().isOffline()){
	    println slave.name;
  }
}
println "master"
`
const upgradeStackScript = `
set +x
TEMPDIR=$(mktemp -d .r_cicd_stacks.XXXX) && cd $TEMPDIR

R_UPGRADESTACK_ENDPOINT=%s
R_UPGRADESTACK_ACCESSKEY=%s
R_UPGRADESTACK_SECRETKEY=%s
R_UPGRADESTACK_STACKNAME=%s
rancher --url $R_UPGRADESTACK_ENDPOINT --access-key $R_UPGRADESTACK_ACCESSKEY --secret-key $R_UPGRADESTACK_SECRETKEY export $R_UPGRADESTACK_STACKNAME

cd $R_UPGRADESTACK_STACKNAME
cat>new-docker-compose.yml<<EOF
%s
EOF
cat>new-rancher-compose.yml<<EOF
%s
EOF
#merge yaml file
mergeyaml -o new-docker-compose.yml new-docker-compose.yml docker-compose.yml 
mergeyaml -o new-rancher-compose.yml new-rancher-compose.yml rancher-compose.yml 
#cat new-docker-compose.yml
#cat new-rancher-compose.yml
rancher --url $R_UPGRADESTACK_ENDPOINT --access-key $R_UPGRADESTACK_ACCESSKEY --secret-key $R_UPGRADESTACK_SECRETKEY up --force-upgrade --confirm-upgrade --pull --file new-docker-compose.yml --rancher-file new-docker-compose.yml -d

rm -r ../../$TEMPDIR

#check stack upgrade
checkSvc()
{
	SvcStatus=$(rancher --url $R_UPGRADESTACK_ENDPOINT --access-key $R_UPGRADESTACK_ACCESSKEY --secret-key $R_UPGRADESTACK_SECRETKEY ps --format "{{.Service.Id}} {{.Stack.Name}} {{.Service.Name}} {{.Service.Transitioning}} {{.Service.TransitioningMessage}}")
	if [ $? -ne 0 ]; then
		echo "upgrade stack $R_UPGRADESTACK_STACKNAME fail: $SvcStatus"
		exit 1
	fi 

	ErrorSvcCount=$(echo "$SvcStatus"|awk '$4=="error" {print $1}'|wc -l);
	if [ $ErrorSvcCount -ne 0 ]; then
		echo "$SvcStatus"|awk '$4=="error" {print "upgrade service ",$1," fail:",}'|cut -f2,5-
		exit 1
	fi
	UpgradingSvcCount=$(echo "$SvcStatus"|awk '$4=="yes" {print $1}'|wc -l);
	if [ $UpgradingSvcCount -ne 0 ]; then
		return 1
	fi
	#upgrade success
	return 0
}

for i in {1..36}
do
	checkSvc;
	if [ $? -eq 0 ]; then
		echo "upgrade stack $R_UPGRADESTACK_STACKNAME success."
		exit 0
	elif [ $? -ne 0 ]; then
		sleep 5
	fi
done

echo "upgrade stack $R_UPGRADESTACK_STACKNAME time out."
exit 1
`

var ErrConfigItemNotFound = errors.New("Jenkins configuration not fount")
var ErrJenkinsTemplateNotVaild = errors.New("Jenkins template folder path is not vaild")
var jenkinsConfLock = &sync.RWMutex{}

func (j jenkinsConfig) Set(key, value string) {
	jenkinsConfLock.Lock()
	defer jenkinsConfLock.Unlock()
	j[key] = value
}

func (j jenkinsConfig) Get(key string) (string, error) {
	jenkinsConfLock.RLock()
	defer jenkinsConfLock.RUnlock()
	if value, ok := j[key]; ok {
		return value, nil
	}
	return "", ErrConfigItemNotFound
}

var JenkinsConfig = jenkinsConfig{
	CreateJobURI:                 "/createItem",
	UpdateJobURI:                 "/job/%s/config.xml",
	DeleteBuildURI:               "/job/%s/lastBuild/doDelete", //purge-job-history/doPurge",
	GetCrumbURI:                  "/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)",
	JenkinsJobBuildURI:           "/job/%s/build",
	JenkinsJobBuildWithParamsURI: "/job/%s/buildWithParameters",
	JenkinsJobInfoURI:            "/job/%s/api/json",
	JenkinsBuildInfoURI:          "/job/%s/lastBuild/api/json",
	JenkinsBuildLogURI:           "/job/%s/lastBuild/timestamps/?elapsed=HH'h'mm'm'ss's'S'ms'&appendLog",
	ScriptURI:                    "/scriptText",
}
