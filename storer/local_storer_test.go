package storer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

var S Storer = InitLocalStorer("/var/tmp/pipelines")

//var S Storer = InitGitStorer("/var/tmp/pipelines", "", "master")

func TestSavePipelineFile(t *testing.T) {
	path := filepath.Join(BasePipelinePath, "testpipeline")
	defer os.RemoveAll(path)
	err := S.SavePipelineFile("testpipeline", "testcontent")
	CheckFatal(t, err)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected pipelineFile saved!")
	}
	b, _ := ioutil.ReadFile(filepath.Join(path, strconv.Itoa(S.GetLatestVersion("testpipeline")), "pipeline.yaml"))
	if string(b) != "testcontent" {
		t.Fatalf("expected \"testcontent\" but get \"%v\"!", string(b))
	}
}

func TestReadLatestPipelineFile(t *testing.T) {
	path := filepath.Join(BasePipelinePath, "testpipeline")
	defer os.RemoveAll(path)
	S.SavePipelineFile("testpipeline", "forreadcontent")
	content, err := S.ReadLatestPipelineFile("testpipeline")
	CheckFatal(t, err)

	if content != "forreadcontent" {
		t.Fatalf("expected \"forreadcontent\" but get \"%v\"!", content)
	}
}

func TestReadPipelineFile(t *testing.T) {
	path := filepath.Join(BasePipelinePath, "testpipeline")
	defer os.RemoveAll(path)
	S.SavePipelineFile("testpipeline", "content0")
	S.SavePipelineFile("testpipeline", "content1")

	content, err := S.ReadPipelineFile("testpipeline", "0")
	CheckFatal(t, err)

	if content != "content0" {
		t.Fatalf("expected \"content0\" but get \"%v\"!", content)
	}
}

func TestSaveLogFile(t *testing.T) {
	defer os.RemoveAll(filepath.Join(BasePipelinePath, "testp"))
	S.SavePipelineFile("testp", "plcontent")
	err := S.SaveLogFile("testp", "0", "stageA", "stepB", "I'm logs")
	CheckFatal(t, err)
	b, _ := ioutil.ReadFile(filepath.Join(BasePipelinePath, "testp", "logs", "0", "stageA_stepB.log"))
	if string(b) != "I'm logs" {
		t.Fatalf("expected \"I'm logs\" but get \"%v\"!", string(b))
	}
}

func TestReadLogFile(t *testing.T) {
	defer os.RemoveAll(filepath.Join(BasePipelinePath, "testp"))
	S.SavePipelineFile("testp", "plcontent")
	S.SaveLogFile("testp", "0", "stageC", "stepD", "I'm logs")
	content, err := S.ReadLogFile("testp", "0", "stageC", "stepD")
	CheckFatal(t, err)
	if content != "I'm logs" {
		t.Fatalf("expected \"I'm logs\" but get \"%v\"!", content)
	}
}

func CheckFatal(t *testing.T, err error) {
	if err == nil {
		return
	}

	// The failure happens at wherever we were called, not here
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		t.Fatalf("Unable to get caller")
	}
	t.Fatalf("Fail at %v:%v; %v", file, line, err)
}
