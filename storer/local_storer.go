package storer

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const (
	LocalStorerName = "local"
)

type LocalStorer struct {
}

func init() {
	if _, err := os.Stat(BasePipelinePath); os.IsNotExist(err) {
		err := os.MkdirAll(BasePipelinePath, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func (l *LocalStorer) GetName() string {
	return LocalStorerName
}

//SavePipelineFile save pipeline file with the content to a new version folder in the path
func (l *LocalStorer) SavePipelineFile(pipelinePath string, content string) error {
	path := filepath.Join(BasePipelinePath, pipelinePath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0755)
		if err != nil {
			return err
		}
	}
	//generate current pipeline file version
	currentVersion := l.GetLatestVersion(pipelinePath) + 1

	err := os.Mkdir(filepath.Join(path, strconv.Itoa(currentVersion)), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(path, strconv.Itoa(currentVersion), "pipeline.yaml"), []byte(content), 0755)
	if err != nil {
		return err
	}
	return nil
}

//ReadPipelineFile read pipeline file in the path with specific version
func (l *LocalStorer) ReadPipelineFile(pipelinePath string, version string) (string, error) {
	path := filepath.Join(BasePipelinePath, pipelinePath, version, "pipeline.yaml")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

//ReadPipelineFile read pipeline file in the path with specific version
func (l *LocalStorer) ReadLatestPipelineFile(pipelinePath string) (string, error) {
	version := l.GetLatestVersion(pipelinePath)
	if version == -1 {
		return "", errors.New("No related pipeline file found")
	}
	return l.ReadPipelineFile(pipelinePath, strconv.Itoa(version))
}

//SaveLogFile saves step log file in "stagename_stepname.log" under pipeline_folder/logs
func (l *LocalStorer) SaveLogFile(pipelinePath string, stageName string, stepName string, content string) error {
	logPath := filepath.Join(BasePipelinePath, pipelinePath, "logs")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0755)
		if err != nil {
			return err
		}
	}

	fname := stageName + "_" + stepName + ".log"
	fPath := filepath.Join(logPath, fname)
	err := ioutil.WriteFile(fPath, []byte(content), 0755)

	return err
}

//ReadLogFile reads log file from pipeline path
func (l *LocalStorer) ReadLogFile(pipelinePath string, stageName string, stepName string) (string, error) {
	fName := stageName + "_" + stepName + ".log"
	fPath := filepath.Join(BasePipelinePath, pipelinePath, "logs", fName)
	b, err := ioutil.ReadFile(fPath)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

//GetLatestVersion gets latest pipeline file version in the pipeline path, return -1 if non valid version exists
func (l *LocalStorer) GetLatestVersion(pipelinePath string) int {
	path := filepath.Join(BasePipelinePath, pipelinePath)
	latestVersion := -1
	files, err := ioutil.ReadDir(path)
	if err != nil {
		panic("getting path fail" + err.Error())
	}
	for _, f := range files {
		i, err := strconv.Atoi(f.Name())
		if err != nil {
			continue
		}
		if i > latestVersion {
			latestVersion = i
		}
	}
	return latestVersion
}
