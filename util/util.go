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

	logrus.Infof("apiconfig:%v", apiConfig)
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

func CreateWebhook(user string, repo string, accesstoken string, webhookUrl string, secret string) error {
	data := user + ":" + accesstoken
	sEnc := base64.StdEncoding.EncodeToString([]byte(data))
	name := "pipeline-" + secret
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

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(hook)
	hc := http.Client{}
	APIURL := fmt.Sprintf("https://api.github.com/repos/%v/%v/hooks", user, repo)
	req, err := http.NewRequest("POST", APIURL, b)

	req.Header.Add("Authorization", sEnc)

	_, err = hc.Do(req)
	return err
}
