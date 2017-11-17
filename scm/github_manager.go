package scm

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/oauth2"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
	"github.com/rancher/pipeline/model"
	"github.com/tomnomnom/linkheader"
	ogithub "golang.org/x/oauth2/github"
)

const githubAPI = "https://api.github.com"
const maxPerPage = "100"

type GithubAccount struct {
	Login       string `json:"login,omitempty"`
	Name        string `json:"name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	HTMLURL     string `json:"html_url,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
}

type GithubManager struct {
}

func (g GithubManager) GetType() string {
	return "github"
}

func (g GithubManager) GetAccount(accessToken string) (*model.GitAccount, error) {
	account, err := getGithubUser(accessToken)
	if err != nil {
		return nil, err
	}
	account.AccessToken = accessToken
	return toAccount(account), nil
}

func (g GithubManager) GetRepos(account *model.GitAccount) (interface{}, error) {
	if account == nil {
		return nil, fmt.Errorf("empty account")
	}
	accessToken := account.AccessToken
	return getGithubRepos(accessToken)
}

func (g GithubManager) OAuth(redirectURL string, clientID string, clientSecret string, code string) (*model.GitAccount, error) {

	logrus.Debugf("github oauth get vars:%v,%v,%v,%v", redirectURL, clientID, clientSecret, code)
	githubOauthConfig := &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes: []string{"repo",
			"admin:repo_hook"},
		Endpoint: ogithub.Endpoint,
	}

	token, err := githubOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		logrus.Errorf("Code exchange failed with '%s'\n", err)
		return nil, err
	} else if token.TokenType != "bearer" || token.AccessToken == "" {
		return nil, fmt.Errorf("Fail to get accesstoken with oauth config")
	}
	logrus.Debugf("get accesstoken:%v", token)
	return g.GetAccount(token.AccessToken)
}
func getGithubUser(githubAccessToken string) (*GithubAccount, error) {

	url := githubAPI + "/user"
	resp, err := getFromGithub(githubAccessToken, url)
	if err != nil {
		logrus.Errorf("Github getGithubUser: GET url %v received error from github, err: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()
	githubAcct := &GithubAccount{}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Github getGithubUser: error reading response, err: %v", err)
		return nil, err
	}

	if err := json.Unmarshal(b, githubAcct); err != nil {
		logrus.Errorf("Github getGithubUser: error unmarshalling response, err: %v", err)
		return nil, err
	}
	return githubAcct, nil
}

func toAccount(gitaccount *GithubAccount) *model.GitAccount {
	if gitaccount == nil {
		return nil
	}
	account := &model.GitAccount{}
	account.AccountType = "github"
	account.AccessToken = gitaccount.AccessToken
	account.AvatarURL = gitaccount.AvatarURL
	account.HTMLURL = gitaccount.HTMLURL
	account.Id = gitaccount.Login
	account.Login = gitaccount.Login
	account.Name = gitaccount.Name
	account.Private = true
	return account
}

func getGithubRepos(githubAccessToken string) ([]github.Repository, error) {
	url := githubAPI + "/user/repos"
	var repos []github.Repository
	responses, err := paginateGithub(githubAccessToken, url)
	if err != nil {
		logrus.Errorf("Github getGithubRepos: GET url %v received error from github, err: %v", url, err)
		return repos, err
	}
	for _, response := range responses {
		defer response.Body.Close()
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			logrus.Errorf("Github getUserRepos: error reading response, err: %v", err)
			return nil, err
		}
		var reposObj []github.Repository
		if err := json.Unmarshal(b, &reposObj); err != nil {
			return repos, err
		}
		repos = append(repos, reposObj...)
	}

	return trimRepo(repos), nil
}

//reduce repo data
func trimRepo(repos []github.Repository) []github.Repository {
	trimed := []github.Repository{}
	for _, repo := range repos {
		trimRepo := github.Repository{}
		trimRepo.CloneURL = repo.CloneURL
		trimRepo.Permissions = repo.Permissions
		trimed = append(trimed, trimRepo)
	}
	return trimed
}

func paginateGithub(githubAccessToken string, url string) ([]*http.Response, error) {
	var responses []*http.Response

	response, err := getFromGithub(githubAccessToken, url)
	if err != nil {
		return responses, err
	}
	responses = append(responses, response)
	nextURL := nextGithubPage(response)
	for nextURL != "" {
		response, err = getFromGithub(githubAccessToken, nextURL)
		if err != nil {
			return responses, err
		}
		responses = append(responses, response)
		nextURL = nextGithubPage(response)
	}

	return responses, nil
}

func getFromGithub(githubAccessToken string, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.Error(err)
	}
	client := &http.Client{}
	//set to max 100 per page to reduce query time
	q := req.URL.Query()
	q.Set("per_page", maxPerPage)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", "token "+githubAccessToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36)")
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Received error from github: %v", err)
		return resp, err
	}
	// Check the status code
	switch resp.StatusCode {
	case 200:
	case 201:
	default:
		var body bytes.Buffer
		io.Copy(&body, resp.Body)
		return resp, fmt.Errorf("Request failed, got status code: %d. Response: %s",
			resp.StatusCode, body.Bytes())
	}
	return resp, nil
}

func nextGithubPage(response *http.Response) string {
	header := response.Header.Get("link")

	if header != "" {
		links := linkheader.Parse(header)
		for _, link := range links {
			if link.Rel == "next" {
				return link.URL
			}
		}
	}

	return ""
}

func (g GithubManager) DeleteWebhook(p *model.Pipeline, token string) error {
	logrus.Infof("deletewebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to delete webhook")
	}

	//delete webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.WebHookId > 0 {
			//TODO
			repoUrl := p.Stages[0].Steps[0].Repository
			reg := regexp.MustCompile(".*?github.com/(.*?)/(.*?).git")
			match := reg.FindStringSubmatch(repoUrl)
			if len(match) != 3 {
				logrus.Infof("get match:%v", match)
				logrus.Errorf("error getting user/repo from gitrepoUrl:%v", repoUrl)
				return errors.New(fmt.Sprintf("error getting user/repo from gitrepoUrl:%v", repoUrl))
			}
			user := match[1]
			repo := match[2]
			err := deleteGithubWebhook(user, repo, token, p.WebHookId)
			if err != nil {
				logrus.Errorf("error delete webhook,%v", err)
				return err
			}
			p.WebHookId = 0
		}
	}
	return nil
}

func (g GithubManager) CreateWebhook(p *model.Pipeline, token string, ciWebhookEndpoint string) error {
	logrus.Debugf("createwebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to create webhook")
	}

	//create webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.Stages[0].Steps[0].Webhook {
			repoUrl := p.Stages[0].Steps[0].Repository
			reg := regexp.MustCompile(".*?github.com/(.*?)/(.*?).git")
			match := reg.FindStringSubmatch(repoUrl)
			if len(match) < 3 {
				logrus.Errorf("error getting user/repo from gitrepoUrl:%v", repoUrl)
				return errors.New(fmt.Sprintf("error getting user/repo from gitrepoUrl:%v", repoUrl))
			}
			user := match[1]
			repo := match[2]
			secret := p.WebHookToken
			webhookUrl := fmt.Sprintf("%s&pipelineId=%s", ciWebhookEndpoint, p.Id)
			id, err := createGithubWebhook(user, repo, token, webhookUrl, secret)
			logrus.Debugf("Creating webhook:%v,%v,%v,%v,%v,%v", user, repo, token, webhookUrl, secret, id)
			if err != nil {
				logrus.Errorf("error delete webhook,%v", err)
				return err
			}
			p.WebHookId = id
		}
	}
	return nil
}

func (g GithubManager) VerifyWebhookPayload(p *model.Pipeline, req *http.Request) bool {
	var signature string
	var event_type string
	if signature = req.Header.Get("X-Hub-Signature"); len(signature) == 0 {
		logrus.Errorf("receive github webhook,no signature")
		return false
	}
	if event_type = req.Header.Get("X-GitHub-Event"); len(event_type) == 0 {
		logrus.Errorf("receive github webhook,no event")
		return false
	}

	if event_type == "ping" {
		return true
	}
	if event_type != "push" {
		logrus.Errorf("receive github webhook,not push event")
		return false
	}
	if p == nil {
		return false
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logrus.Errorf("receive github webhook, got error:%v", err)
		return false
	}
	if match := VerifyGithubWebhookSignature([]byte(p.WebHookToken), signature, body); !match {
		logrus.Errorf("receive github webhook, invalid signature")
		return false
	}
	return true
}

func VerifyGithubWebhookSignature(secret []byte, signature string, body []byte) bool {

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
func createGithubWebhook(user string, repo string, accesstoken string, webhookUrl string, secret string) (int, error) {
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

func deleteGithubWebhook(user string, repo string, accesstoken string, id int) error {

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
