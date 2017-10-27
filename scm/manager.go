package scm

import "github.com/rancher/go-rancher/client"

type Manager interface {
	GetRepos(account *Account) (interface{}, error)
	GetAccount(accessToken string) (*Account, error)
	OAuth(redirectURL string, clientID string, clientSecret string, code string) (*Account, error)
}

type Account struct {
	client.Resource
	//private or shared across environment
	Private       bool   `json:"private,omitempty"`
	AccountType   string `json:"accountType,omitempty"`
	RancherUserID string `json:"rancherUserId,omitempty"`
	Status        string `json:"status,omitempty"`

	Login       string `json:"login,omitempty"`
	Name        string `json:"name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	HTMLURL     string `json:"html_url,omitempty"`
	AccessToken string `json:"accessToken,omitempty"`
}
