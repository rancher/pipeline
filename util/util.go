package util

import (
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/pipeline/config"
)

func GetRancherClient() (*client.RancherClient, error) {
	apiConfig := config.GetConfig()

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
