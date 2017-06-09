package pipeline

import (
	"github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var data = `---
name: test1
repository: http://github.com/orangedeng/ui.git
branch: master
target_image: rancher/ui:v0.1
stages:
  - name: stage zero
    need_approve: false
    steps:
    - name: step zero
      image: test/build:v0.1
      command: make
      parameters:
      - "env=dev"
  - name: stage test
    need_approve: false
    steps:
    - name: source code check
      image: test/test:v0.1
      command: echo 'i am test'
    - name: server run test
      image: test/run-bin:v0.1
      command: /startup.sh
    - name: API test 
      image: test/api-test:v0.1
      command: /startup.sh && /api_test.sh
`

type Pipeline struct {
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	//	Version        string   `json:"version,omitempty" yaml:"version,omitempty"`
	Repository  string   `json:"repository,omitempty" yaml:"repository,omitempty"`
	Branch      string   `json:"branch,omitempty" yaml:"branch,omitempty"`
	TargetImage string   `json:"target_image,omitempty" yaml:"target_image,omitempty"`
	File        string   `json:"file"`
	Stages      []*Stage `json:"stages,omitempty" yaml:"stages,omitempty"`
}

type Stage struct {
	Name        string  `json:"name,omitempty" yaml:"name,omitempty"`
	NeedApprove bool    `json:"need_approve,omitempty" yaml:"need_approve,omitempty"`
	Steps       []*Step `json:"steps,omitempty" yaml:"steps,omitempty"`
}

type Step struct {
	Name       string   `json:"name,omitempty" yaml:"name,omitempty"`
	Image      string   `json:"image,omitempty" yaml:"image,omitempty"`
	Command    string   `json:"command,omitempty" yaml:"command,omitempty"`
	Parameters []string `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

func ToDemoPipeline() *Pipeline {
	r := Pipeline{}
	if err := yaml.Unmarshal([]byte(data), &r); err != nil {
		logrus.Error(err)
		return nil
	}
	r.File = data
	return &r
}
