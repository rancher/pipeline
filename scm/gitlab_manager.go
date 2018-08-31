package scm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/Sirupsen/logrus"
	"github.com/google/go-querystring/query"
	"github.com/rancher/pipeline/model"
	"github.com/tomnomnom/linkheader"
	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

//use v4 endpoint for compatibility
const gitlabAPI = "%s%s/api/v4"

type GitlabManager struct {
	host   string
	scheme string
}

func (g GitlabManager) Config(setting *model.SCMSetting) model.SCManager {
	if setting.Scheme != "" {
		g.scheme = setting.Scheme
	} else {
		g.scheme = "https://"
	}
	if setting.HostName != "" {
		g.host = setting.HostName
	} else {
		g.host = "gitlab.com"
	}

	return g
}

func (g GitlabManager) GetType() string {
	return "gitlab"
}

func (g GitlabManager) GetAccount(accessToken string) (*model.GitAccount, error) {
	account, err := g.getGitlabUser(accessToken)
	if err != nil {
		return nil, err
	}
	gitAccount := toGitlabAccount(account)
	gitAccount.AccessToken = accessToken
	return gitAccount, nil
}

func (g GitlabManager) GetRepos(account *model.GitAccount) ([]*model.GitRepository, error) {
	if account == nil {
		return nil, fmt.Errorf("empty account")
	}
	accessToken := account.AccessToken
	return g.getGitlabRepos(accessToken)
}

func (g GitlabManager) OAuth(redirectURL string, clientID string, clientSecret string, code string) (*model.GitAccount, error) {

	gitlabOauthConfig := &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"api"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s%s/oauth/authorize", g.scheme, g.host),
			TokenURL: fmt.Sprintf("%s%s/oauth/token", g.scheme, g.host),
		},
	}

	token, err := gitlabOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		logrus.Errorf("Code exchange failed with '%s'\n", err)
		return nil, err
	} else if token.TokenType != "bearer" || token.AccessToken == "" {
		return nil, fmt.Errorf("Fail to get accesstoken with oauth config")
	}
	return g.GetAccount(token.AccessToken)
}

func (g GitlabManager) getGitlabUser(gitlabAccessToken string) (*gitlab.User, error) {

	url := fmt.Sprintf(gitlabAPI+"/user", g.scheme, g.host)
	resp, err := getFromGitlab(gitlabAccessToken, url)
	if err != nil {
		logrus.Errorf("Gitlab getGitlabUser: GET url %v received error from gitlab, err: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()
	gitlabAcct := &gitlab.User{}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Gitlab getGitlabUser: error reading response, err: %v", err)
		return nil, err
	}

	if err := json.Unmarshal(b, gitlabAcct); err != nil {
		logrus.Errorf("Gitlab getGitlabUser: error unmarshalling response, err: %v", err)
		return nil, err
	}
	return gitlabAcct, nil
}

func toGitlabAccount(gitaccount *gitlab.User) *model.GitAccount {
	if gitaccount == nil {
		return nil
	}
	account := &model.GitAccount{}
	account.AccountType = "gitlab"
	account.AvatarURL = gitaccount.AvatarURL
	account.HTMLURL = gitaccount.WebsiteURL
	account.Id = "gitlab:" + gitaccount.Username
	account.Login = gitaccount.Username
	account.Name = gitaccount.Name
	account.Private = false
	return account
}

func (g GitlabManager) getGitlabRepos(gitlabAccessToken string) ([]*model.GitRepository, error) {
	url := fmt.Sprintf(gitlabAPI+"/projects?membership=true", g.scheme, g.host)
	var repos []gitlab.Project
	responses, err := paginateGitlab(gitlabAccessToken, url)
	if err != nil {
		logrus.Errorf("Gitlab getGitlabRepos: GET url %v received error from gitlab, err: %v", url, err)
		return nil, err
	}
	for _, response := range responses {
		defer response.Body.Close()
		b, err := ioutil.ReadAll(response.Body)
		if err != nil {
			logrus.Errorf("Gitlab getUserRepos: error reading response, err: %v", err)
			return nil, err
		}
		var reposObj []gitlab.Project
		if err := json.Unmarshal(b, &reposObj); err != nil {
			return nil, err
		}
		repos = append(repos, reposObj...)
	}

	return g.toGitRepo(repos), nil
}

//reduce repo data
func (g GitlabManager) toGitRepo(repos []gitlab.Project) []*model.GitRepository {
	result := []*model.GitRepository{}
	for _, repo := range repos {
		r := &model.GitRepository{}
		r.CloneURL = repo.HTTPURLToRepo
		r.Permissions = map[string]bool{}
		accessLevel := getAccessLevel(repo)
		if accessLevel >= 20 {
			// 20 for 'Reporter' level
			r.Permissions["pull"] = true
		}
		if accessLevel >= 30 {
			// 30 for 'Developer' level
			r.Permissions["push"] = true
		}
		if accessLevel >= 40 {
			// 40 for 'Master' level and 50 for 'Owner' level
			r.Permissions["admin"] = true
		}
		result = append(result, r)
	}
	return result
}

func getAccessLevel(repo gitlab.Project) int {
	accessLevel := 0
	if repo.Permissions == nil {
		return accessLevel
	}
	if repo.Permissions.ProjectAccess != nil && int(repo.Permissions.ProjectAccess.AccessLevel) > accessLevel {
		accessLevel = int(repo.Permissions.ProjectAccess.AccessLevel)
	}
	if repo.Permissions.GroupAccess != nil && int(repo.Permissions.GroupAccess.AccessLevel) > accessLevel {
		accessLevel = int(repo.Permissions.GroupAccess.AccessLevel)
	}
	return accessLevel
}

func paginateGitlab(gitlabAccessToken string, url string) ([]*http.Response, error) {
	var responses []*http.Response

	response, err := getFromGitlab(gitlabAccessToken, url)
	if err != nil {
		return responses, err
	}
	responses = append(responses, response)
	nextURL := nextGitlabPage(response)
	for nextURL != "" {
		response, err = getFromGitlab(gitlabAccessToken, nextURL)
		if err != nil {
			return responses, err
		}
		responses = append(responses, response)
		nextURL = nextGitlabPage(response)
	}

	return responses, nil
}

func getFromGitlab(gitlabAccessToken string, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logrus.Error(err)
	}
	client := &http.Client{}
	//set to max 100 per page to reduce query time
	q := req.URL.Query()
	q.Set("per_page", maxPerPage)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", "Bearer "+gitlabAccessToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.103 Safari/537.36)")
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Received error from gitlab: %v", err)
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

func nextGitlabPage(response *http.Response) string {
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

func (g GitlabManager) DeleteWebhook(p *model.Pipeline, token string) error {
	logrus.Debugf("deletewebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to delete webhook")
	}

	//delete webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.WebHookId > 0 {
			user, repo, err := getUserRepoFromURL(p.Stages[0].Steps[0].Repository)
			if err != nil {
				return nil
			}
			if err := g.deleteGitlabWebhook(user, repo, token, p.WebHookId); err != nil {
				logrus.Errorf("error delete webhook,%v", err)
				return err
			}
			p.WebHookId = 0
		}
	}
	return nil
}

func (g GitlabManager) CreateWebhook(p *model.Pipeline, token string, ciWebhookEndpoint string) error {
	logrus.Debugf("createwebhook for pipeline:%v", p.Id)
	if p == nil {
		return errors.New("empty pipeline to create webhook")
	}

	//create webhook
	if len(p.Stages) > 0 && len(p.Stages[0].Steps) > 0 {
		if p.Stages[0].Steps[0].Webhook {
			user, repo, err := getUserRepoFromURL(p.Stages[0].Steps[0].Repository)
			if err != nil {
				return nil
			}
			secret := p.WebHookToken
			webhookUrl := fmt.Sprintf("%s&pipelineId=%s", ciWebhookEndpoint, p.Id)

			id, err := g.createGitlabWebhook(user, repo, token, webhookUrl, secret)
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

func (g GitlabManager) VerifyWebhookPayload(p *model.Pipeline, req *http.Request) bool {
	var signature string
	var event_type string
	if signature = req.Header.Get("X-Gitlab-Token"); len(signature) == 0 {
		logrus.Warningf("receive gitlab webhook, but got no token")
		return false
	}
	if event_type = req.Header.Get("X-Gitlab-Event"); len(event_type) == 0 {
		logrus.Warningf("receive gitlab webhook, but got no event")
		return false
	}

	if event_type != "Push Hook" {
		logrus.Warningf("receive gitlab webhook '%s' event, expected push hook event", event_type)
		return false
	}
	if p == nil {
		return false
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logrus.Warningf("receive gitlab webhook, got error:%v", err)
		return false
	}
	if p.WebHookToken != signature {
		logrus.Warning("receive gitlab webhook, invalid token")
		return false
	}
	//check branch
	payload := map[string]interface{}{}
	logrus.Debugf("gitlab webhook got payload:\n%v", string(body))
	if err := json.Unmarshal(body, &payload); err != nil {
		logrus.Error("fail to parse github webhook payload,err:%v", err)
		return false
	}
	if payload["ref"] != "refs/heads/"+p.Stages[0].Steps[0].Branch {
		logrus.Warningf("receive gitlab webhook, branch not match:%v,%v", payload["ref"], p.Stages[0].Steps[0].Branch)
		return false
	}
	return true
}

func VerifyGitlabWebhookSignature(secret []byte, signature string, body []byte) bool {
	return false
}

//create webhook,return id of webhook
func (g GitlabManager) createGitlabWebhook(user string, repo string, accesstoken string, webhookUrl string, secret string) (int, error) {

	project := url.QueryEscape(user + "/" + repo)
	client := http.Client{}
	APIURL := fmt.Sprintf(gitlabAPI+"/projects/%s/hooks", g.scheme, g.host, project)
	req, err := http.NewRequest("POST", APIURL, nil)

	opt := &gitlab.AddProjectHookOptions{
		PushEvents: gitlab.Bool(true),
		URL:        gitlab.String(webhookUrl),
		EnableSSLVerification: gitlab.Bool(false),
		Token: gitlab.String(secret),
	}
	q, err := query.Values(opt)
	if err != nil {
		return 0, err
	}
	logrus.Debugf("gitlab hook to create:%v", opt)
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", "Bearer "+accesstoken)

	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	logrus.Debugf("respData:%v", string(respData))
	if resp.StatusCode > 399 {
		return -1, errors.New(string(respData))
	}
	hook := gitlab.ProjectHook{}
	err = json.Unmarshal(respData, &hook)
	if err != nil {
		return -1, err
	}
	return hook.ID, err
}

func (g GitlabManager) deleteGitlabWebhook(user string, repo string, accesstoken string, id int) error {
	client := http.Client{}
	project := url.QueryEscape(user + "/" + repo)
	APIURL := fmt.Sprintf(gitlabAPI+"/projects/%s/hooks/%d", g.scheme, g.host, project, id)
	req, err := http.NewRequest("DELETE", APIURL, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+accesstoken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode > 399 {
		return errors.New(string(respData))
	}
	return err
}
