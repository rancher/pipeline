package jenkins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/rancher/pipeline/config"
)

var (
	ErrCreateJobFail    = errors.New("Create Job fail")
	ErrBuildJobFail     = errors.New("Build Job fail")
	ErrGetBuildInfoFail = errors.New("Get Build Info fail")
	ErrGetJobInfoFail   = errors.New("Get Job Info fail")
)

func InitJenkins() {
	var jenkinsServerAddress, user, token string
	jenkinsServerAddress = config.Config.JenkinsAddress //context.String("jenkins_address")
	user = config.Config.JenkinsUser                    //context.String("jenkins_user")
	token = config.Config.JenkinsToken                  //context.String("jenkins_token")
	/*
		jenkinsTemlpateFolder := context.String("jenkins_config_template")
		if fi, err := os.Stat(jenkinsTemlpateFolder); err != nil {
			logrus.Fatal(errors.Wrapf(err, "jenkins template folder read error"))
		} else {
			if !fi.IsDir() {
				logrus.Fatal(ErrJenkinsTemplateNotVaild)
			}
		}
		JenkinsConfig.Set(JenkinsTemlpateFolder, jenkinsTemlpateFolder)
	*/
	JenkinsConfig.Set(JenkinsServerAddress, jenkinsServerAddress)
	JenkinsConfig.Set(JenkinsUser, user)
	JenkinsConfig.Set(JenkinsToken, token)
	logrus.Info("Connectting to Jenkins...")
	if err := GetCSRF(); err != nil {
		logrus.Fatalf("Error Connectting to Jenkins err:%s", err.Error())
	}
	logrus.Info("Connected to Jenkins")
}

func GetCSRF() error {
	sah, _ := JenkinsConfig.Get(JenkinsServerAddress)
	getCrumbURI, _ := JenkinsConfig.Get(GetCrumbURI)
	user, _ := JenkinsConfig.Get(JenkinsUser)
	token, _ := JenkinsConfig.Get(JenkinsToken)
	getCrumbURL, err := url.Parse(sah + getCrumbURI)
	if err != nil {
		logrus.Error(err)
	}
	req, _ := http.NewRequest(http.MethodGet, getCrumbURL.String(), nil)
	req.SetBasicAuth(user, token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return err
	}
	data, _ := ioutil.ReadAll(resp.Body)
	Crumbs := strings.Split(string(data), ":")
	if len(Crumbs) != 2 {
		return fmt.Errorf("Return Crumbs From Jenkins Error:<%s>", err.Error())
	}
	JenkinsConfig.Set(JenkinsCrumbHeader, Crumbs[0])
	JenkinsConfig.Set(JenkinsCrumb, Crumbs[1])
	return nil
}

func CreateJob(jobname string, content []byte) error {
	sah, _ := JenkinsConfig.Get(JenkinsServerAddress)
	createJobURI, _ := JenkinsConfig.Get(CreateJobURI)
	user, _ := JenkinsConfig.Get(JenkinsUser)
	token, _ := JenkinsConfig.Get(JenkinsToken)
	CrumbHeader, _ := JenkinsConfig.Get(JenkinsCrumbHeader)
	Crumb, _ := JenkinsConfig.Get(JenkinsCrumb)

	//url part
	createJobURL, err := url.Parse(sah + createJobURI)
	if err != nil {
		logrus.Error(err)
		return err
	}
	qry := createJobURL.Query()
	qry.Add("name", jobname)
	createJobURL.RawQuery = qry.Encode()
	//send request part
	req, _ := http.NewRequest(http.MethodPost, createJobURL.String(), bytes.NewReader(content))
	req.Header.Add(CrumbHeader, Crumb)
	req.Header.Set("Content-Type", "application/xml")
	req.SetBasicAuth(user, token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return err
	}
	// data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		data, _ := ioutil.ReadAll(resp.Body)
		println(string(data))
		return ErrCreateJobFail
	}
	return nil
}

func BuildJob(jobname string, params map[string]string) (string, error) {
	sah, _ := JenkinsConfig.Get(JenkinsServerAddress)
	buildURI, _ := JenkinsConfig.Get(JenkinsJobBuildURI)
	buildURI = fmt.Sprintf(buildURI, jobname)
	buildWithParamsURI, _ := JenkinsConfig.Get(JenkinsJobBuildWithParamsURI)
	buildWithParamsURI = fmt.Sprintf(buildWithParamsURI, jobname)
	user, _ := JenkinsConfig.Get(JenkinsUser)
	token, _ := JenkinsConfig.Get(JenkinsToken)
	CrumbHeader, _ := JenkinsConfig.Get(JenkinsCrumbHeader)
	Crumb, _ := JenkinsConfig.Get(JenkinsCrumb)

	withParams := false
	if len(params) > 0 {
		withParams = true
	}
	var targetURL *url.URL
	var err error
	if withParams {
		targetURL, err = url.Parse(sah + buildWithParamsURI)
	} else {
		targetURL, err = url.Parse(sah + buildURI)
	}
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	req, _ := http.NewRequest(http.MethodPost, targetURL.String(), nil)

	req.Header.Add(CrumbHeader, Crumb)
	req.SetBasicAuth(user, token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	if resp.StatusCode != 201 {
		logrus.Error(ErrBuildJobFail)
		return "", ErrBuildJobFail
	}
	logrus.Infof("job queue is %s", resp.Header.Get("location"))
	return "", nil
}

func GetBuildInfo(jobname string) (*JenkinsBuildInfo, error) {
	sah, _ := JenkinsConfig.Get(JenkinsServerAddress)
	buildInfoURI, _ := JenkinsConfig.Get(JenkinsBuildInfoURI)
	buildInfoURI = fmt.Sprintf(buildInfoURI, jobname)
	user, _ := JenkinsConfig.Get(JenkinsUser)
	token, _ := JenkinsConfig.Get(JenkinsToken)
	CrumbHeader, _ := JenkinsConfig.Get(JenkinsCrumbHeader)
	Crumb, _ := JenkinsConfig.Get(JenkinsCrumb)

	var targetURL *url.URL
	var err error
	targetURL, err = url.Parse(sah + buildInfoURI)
	logrus.Infof("targetURL is :%v", targetURL)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodPost, targetURL.String(), nil)

	req.Header.Add(CrumbHeader, Crumb)
	req.SetBasicAuth(user, token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Infof("response code is :%v", resp.StatusCode)
	if resp.StatusCode != 200 {
		logrus.Error(ErrGetBuildInfoFail)
		return nil, ErrGetBuildInfoFail
	}
	buildInfo := &JenkinsBuildInfo{}
	respBytes, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(respBytes, buildInfo)
	if err != nil {
		return nil, err
	}

	logrus.Infof("build info is %v", buildInfo)
	return buildInfo, nil

}

func GetJobInfo(jobname string) (*JenkinsJobInfo, error) {
	sah, _ := JenkinsConfig.Get(JenkinsServerAddress)
	jobInfoURI, _ := JenkinsConfig.Get(JenkinsJobInfoURI)
	jobInfoURI = fmt.Sprintf(jobInfoURI, jobname)
	user, _ := JenkinsConfig.Get(JenkinsUser)
	token, _ := JenkinsConfig.Get(JenkinsToken)
	CrumbHeader, _ := JenkinsConfig.Get(JenkinsCrumbHeader)
	Crumb, _ := JenkinsConfig.Get(JenkinsCrumb)

	var targetURL *url.URL
	var err error
	targetURL, err = url.Parse(sah + jobInfoURI)
	logrus.Infof("targetURL is :%v", targetURL)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	req, _ := http.NewRequest(http.MethodGet, targetURL.String(), nil)

	req.Header.Add(CrumbHeader, Crumb)
	req.SetBasicAuth(user, token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	logrus.Infof("response code is :%v", resp.StatusCode)
	if resp.StatusCode != 200 {
		logrus.Error(ErrGetJobInfoFail)
		return nil, ErrGetJobInfoFail
	}
	jobInfo := &JenkinsJobInfo{}
	respBytes, err := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(respBytes, jobInfo)
	if err != nil {
		return nil, err
	}

	logrus.Infof("build info is %v", jobInfo)
	return jobInfo, nil

}

func GetBuildRawOutput(jobname string) (string, error) {
	sah, _ := JenkinsConfig.Get(JenkinsServerAddress)
	buildRawOutputURI, _ := JenkinsConfig.Get(JenkinsBuildLogURI)
	buildRawOutputURI = fmt.Sprintf(buildRawOutputURI, jobname)
	user, _ := JenkinsConfig.Get(JenkinsUser)
	token, _ := JenkinsConfig.Get(JenkinsToken)
	CrumbHeader, _ := JenkinsConfig.Get(JenkinsCrumbHeader)
	Crumb, _ := JenkinsConfig.Get(JenkinsCrumb)

	var targetURL *url.URL
	var err error
	targetURL, err = url.Parse(sah + buildRawOutputURI)
	logrus.Infof("targetURL is :%v", targetURL)
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	req, _ := http.NewRequest(http.MethodGet, targetURL.String(), nil)

	req.Header.Add(CrumbHeader, Crumb)
	req.SetBasicAuth(user, token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	logrus.Infof("response code is :%v", resp.StatusCode)
	if resp.StatusCode != 200 {
		logrus.Error(ErrGetJobInfoFail)
		return "", ErrGetJobInfoFail
	}
	respBytes, err := ioutil.ReadAll(resp.Body)

	logrus.Infof("build info is %v", string(respBytes))
	return string(respBytes), nil

}
