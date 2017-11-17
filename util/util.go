package util

import (
	"errors"
	"math/rand"
	"net/http"
	"regexp"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/config"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

/**
 * Parses url with the given regular expression and returns the
 * group values defined in the expression.
 *
 */
func GetParams(regEx, url string) (paramsMap map[string]string) {

	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(url)

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

func GetRancherClient() (*client.RancherClient, error) {
	apiConfig := config.Config
	apiUrl := apiConfig.CattleUrl
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

func GetCurrentUser(cookies []*http.Cookie) (string, error) {

	client := &http.Client{}

	requestURL := config.Config.CattleUrl + "/accounts"

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		logrus.Infof("Cannot connect to the rancher server. Please check the rancher server URL")
		return "", err
	}

	//req.SetBasicAuth(config.Config.CattleAccessKey, config.Config.CattleSecretKey)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Infof("Cannot connect to the rancher server. Please check the rancher server URL")
		return "", err
	}
	defer resp.Body.Close()
	userid := resp.Header.Get("X-Api-User-Id")
	if userid == "" {
		logrus.Infof("Cannot get userid")
		err := errors.New("Forbidden")
		return "Forbidden", err

	}
	return userid, nil
}
