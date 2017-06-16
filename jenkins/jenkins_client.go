package jenkins

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"os"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var (
	ErrCreateJobFail = errors.New("Create Job fail")
	ErrBuildJobFail  = errors.New("Build Job fail")
)

func InitJenkins(context *cli.Context) {
	var jenkinsServerAddress, user, token string
	jenkinsServerAddress = context.String("jenkins_address")
	user = context.String("jenkins_user")
	token = context.String("jenkins_token")
	jenkinsTemlpateFolder := context.String("jenkins_config_template")
	if fi, err := os.Stat(jenkinsTemlpateFolder); err != nil {
		logrus.Fatal(errors.Wrapf(err, "jenkins template folder read error"))
	} else {
		if !fi.IsDir() {
			logrus.Fatal(ErrJenkinsTemplateNotVaild)
		}
	}
	JenkinsConfig.Set(JenkinsTemlpateFolder, jenkinsTemlpateFolder)
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
