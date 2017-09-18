package restfulserver

import (
	"net/http"

	"golang.org/x/oauth2"

	"github.com/Sirupsen/logrus"
	"golang.org/x/oauth2/github"
)

const oauthStateString = "random"

func (s *Server) GithubLogin(rw http.ResponseWriter, req *http.Request) error {

	githubOauthConfig, err := getGithubOauthConfig()
	if err != nil {
		return err
	}
	url := githubOauthConfig.AuthCodeURL(oauthStateString)
	http.Redirect(rw, req, url, http.StatusTemporaryRedirect)

	return nil
}

func (s *Server) GithubAuthorize(rw http.ResponseWriter, req *http.Request) error {
	state := req.FormValue("state")
	if state != oauthStateString {
		logrus.Errorf("invalid oauth state, expected '%s', got '%s'\n", oauthStateString, state)
		http.Redirect(rw, req, "/", http.StatusTemporaryRedirect)
		return nil
	}
	githubOauthConfig, err := getGithubOauthConfig()
	if err != nil {
		return err
	}
	code := req.FormValue("code")
	token, err := githubOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		logrus.Errorf("Code exchange failed with '%s'\n", err)
		http.Redirect(rw, req, "/", http.StatusTemporaryRedirect)
		return nil
	}
	if err := saveGithubToken(token.AccessToken); err != nil {
		return err
	}

	return nil
}

func getGithubOauthConfig() (*oauth2.Config, error) {
	setting, err := GetPipelineSetting()
	if err != nil {
		return nil, err
	}
	githubOauthConfig := &oauth2.Config{
		RedirectURL:  setting.GithubRedirectURL,
		ClientID:     setting.GithubClientID,
		ClientSecret: setting.GithubClientSecret,
		Scopes: []string{"repo",
			"admin:repo_hook"},
		Endpoint: github.Endpoint,
	}
	return githubOauthConfig, nil
}

func saveGithubToken(token string) error {
	setting, err := GetPipelineSetting()
	if err != nil {
		return err
	}
	exist := false
	for _, t := range setting.GithubTokens {
		if t == token {
			exist = true
		}
	}
	if !exist {
		setting.GithubTokens = append(setting.GithubTokens, token)
	}
	return CreateOrUpdatePipelineSetting(setting)
}
