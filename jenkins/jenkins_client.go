package jenkins

import "net/http"
import "net/url"
import "io/ioutil"
import "bytes"
import "github.com/Sirupsen/logrus"
import "strings"

const user = "admin"
const token = "9823967c90e16797c5c8e7fe5c066979"
const jenkinsAddress = "192.168.99.100:8888"
const jenkinsSchema = "http"
const createJobURI = "/createItem"
const getCrumbURI = "/crumbIssuer/api/xml?xpath=concat(//crumbRequestField,\":\",//crumb)"

var crumbHeader = ""
var crumb = ""

func callJenkins(req http.Request) error {
	return nil
}

func GetCSRF() error {
	println("get in")
	sah := jenkinsSchema + `://` + jenkinsAddress
	println(sah)
	getCrumbURL, err := url.Parse(sah + getCrumbURI)
	if err != nil {
		logrus.Error(err)
	}
	println(getCrumbURL)
	req, _ := http.NewRequest(http.MethodGet, getCrumbURL.String(), nil)
	req.SetBasicAuth(user, token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return err
	}
	data, _ := ioutil.ReadAll(resp.Body)
	println(string(data))
	crumbHeader = strings.Split(string(data), ":")[0]
	crumb = strings.Split(string(data), ":")[1]
	return nil
}

func CreateJob(jobname string) error {
	//url part
	println("get in")
	sah := jenkinsSchema + `://` + jenkinsAddress
	println(sah)
	createJobURL, err := url.Parse(sah + createJobURI)
	if err != nil {
		logrus.Error(err)
		return err
	}
	println(createJobURL.String())
	qry := createJobURL.Query()
	qry.Add("name", jobname)
	createJobURL.RawQuery = qry.Encode()
	println("query raw is ", qry.Encode())
	//body part
	body, _ := ioutil.ReadFile("jenkins/example_job.xml")
	//send request part
	req, _ := http.NewRequest(http.MethodPost, createJobURL.String(), bytes.NewReader(body))
	req.Header.Add("Jenkins-Crumb", crumb)
	req.Header.Set("Content-Type", "application/xml")
	req.SetBasicAuth(user, token)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error(err)
		return err
	}
	data, _ := ioutil.ReadAll(resp.Body)
	println(string(data))
	//
	return nil
}
