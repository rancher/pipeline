package util

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/config"
)

func GetRancherClient() (*client.RancherClient, error) {
	apiConfig := config.Config
	apiUrl := apiConfig.CattleUrl //http://ip:port/v2
	accessKey := apiConfig.CattleAccessKey
	secretKey := apiConfig.CattleSecretKey

	apiClient, err := client.NewRancherClient(&client.ClientOpts{
		Url:       apiUrl,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})
	if err != nil {
		return nil, err
	}
	if apiClient == nil {
		return nil, errors.New("fail to get rancherClient")
	}
	return apiClient, nil
}

func GetProjectId() (string, error) {

	client := &http.Client{}

	requestURL := config.Config.CattleUrl + "/accounts"

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		logrus.Errorf("Cannot connect to the rancher server. Please check the rancher server URL")
		return "", err
	}
	req.SetBasicAuth(config.Config.CattleAccessKey, config.Config.CattleSecretKey)
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Cannot connect to the rancher server. Please check the rancher server URL")
		return "", err
	}
	defer resp.Body.Close()
	projectId := resp.Header.Get("X-Api-Account-Id")
	if projectId == "" {
		logrus.Errorln("Cannot get projectId")
		err := errors.New("Forbidden")
		return "Forbidden", err

	}
	return projectId, nil

}

func VerifyWebhookSignature(secret []byte, signature string, body []byte) bool {

	const signaturePrefix = "sha1="
	const signatureLength = 45 // len(SignaturePrefix) + len(hex(sha1))

	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}

	actual := make([]byte, 20)
	hex.Decode(actual, []byte(signature[5:]))
	computed := hmac.New(sha1.New, secret)
	computed.Write(body)

	return hmac.Equal([]byte(computed.Sum(nil)), actual)
}

//create webhook,return id of webhook
func CreateWebhook(user string, repo string, accesstoken string, webhookUrl string, secret string) (int, error) {
	data := user + ":" + accesstoken
	sEnc := base64.StdEncoding.EncodeToString([]byte(data))
	name := "web"
	active := true
	hook := github.Hook{
		Name:   &name,
		Active: &active,
		Config: make(map[string]interface{}),
		Events: []string{"push"},
	}

	hook.Config["url"] = webhookUrl
	hook.Config["content_type"] = "json"
	hook.Config["secret"] = secret
	hook.Config["insecure_ssl"] = "1"

	logrus.Infof("hook to create:%v", hook)
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(hook)
	hc := http.Client{}
	APIURL := fmt.Sprintf("https://api.github.com/repos/%v/%v/hooks", user, repo)
	req, err := http.NewRequest("POST", APIURL, b)

	req.Header.Add("Authorization", "Basic "+sEnc)

	resp, err := hc.Do(req)
	if err != nil {
		return -1, err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	logrus.Infof("respData:%v", string(respData))
	if resp.StatusCode > 399 {
		return -1, errors.New(string(respData))
	}
	err = json.Unmarshal(respData, &hook)
	if err != nil {
		return -1, err
	}
	return hook.GetID(), err
}

func ListWebhook(user string, repo string, accesstoken string) ([]*github.Hook, error) {
	data := user + ":" + accesstoken
	sEnc := base64.StdEncoding.EncodeToString([]byte(data))
	hc := http.Client{}
	APIURL := fmt.Sprintf("https://api.github.com/repos/%v/%v/hooks", user, repo)
	req, err := http.NewRequest("GET", APIURL, nil)
	if err != nil {
		return nil, err
	}
	var hooks []*github.Hook
	logrus.Infof("get encrpyt:%v", sEnc)
	req.Header.Add("Authorization", "Basic "+sEnc)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	logrus.Infof("get response data:%v", string(respData))
	if resp.StatusCode > 399 {
		return nil, errors.New(string(respData))
	}
	err = json.Unmarshal(respData, &hooks)
	if err != nil {
		return nil, err
	}

	return hooks, nil
}

func GetWebhook(user string, repo string, accesstoken string, id string) (*github.Hook, error) {
	data := user + ":" + accesstoken
	sEnc := base64.StdEncoding.EncodeToString([]byte(data))
	hc := http.Client{}
	APIURL := fmt.Sprintf("https://api.github.com/repos/%v/%v/hooks", user, repo)
	req, err := http.NewRequest("GET", APIURL, nil)
	if err != nil {
		return nil, err
	}
	var hook *github.Hook
	logrus.Infof("get encrpyt:%v", sEnc)
	req.Header.Add("Authorization", "Basic "+sEnc)
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	logrus.Infof("get response data:%v", string(respData))
	if resp.StatusCode > 399 {
		return nil, errors.New(string(respData))
	}
	err = json.Unmarshal(respData, hook)
	if err != nil {
		return nil, err
	}
	return hook, nil
}

func DeleteWebhook(user string, repo string, accesstoken string, id int) error {

	logrus.Infof("deleting webhook:%v,%v,%v,%v", user, repo, accesstoken, id)
	data := user + ":" + accesstoken
	sEnc := base64.StdEncoding.EncodeToString([]byte(data))
	hc := http.Client{}
	APIURL := fmt.Sprintf("https://api.github.com/repos/%v/%v/hooks/%v", user, repo, id)
	req, err := http.NewRequest("DELETE", APIURL, nil)
	if err != nil {
		return err
	}
	logrus.Infof("get encrpyt:%v", sEnc)
	req.Header.Add("Authorization", "Basic "+sEnc)
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode > 399 {
		return errors.New(string(respData))
	}
	logrus.Infof("after delete,%v,%v", string(respData))
	return err
}
