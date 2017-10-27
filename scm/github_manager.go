package scm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-github/github"
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

func (g *GithubManager) GetAccount(accessToken string) (*Account, error) {
	account, err := getGithubUser(accessToken)
	if err != nil {
		return nil, err
	}
	account.AccessToken = accessToken
	return toAccount(account), nil
}

func (g *GithubManager) GetRepos(account *Account) (interface{}, error) {
	if account == nil {
		return nil, fmt.Errorf("empty account")
	}
	accessToken := account.AccessToken
	return getGithubRepos(accessToken)
}

func (g *GithubManager) OAuth(redirectURL string, clientID string, clientSecret string, code string) (*Account, error) {

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

func toAccount(gitaccount *GithubAccount) *Account {
	if gitaccount == nil {
		return nil
	}
	account := &Account{}
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
