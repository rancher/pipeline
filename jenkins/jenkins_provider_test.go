package jenkins

import (
	"testing"

	"github.com/rancher/pipeline/config"
	"github.com/rancher/pipeline/pipeline"
)

func init() {
	config.Config.JenkinsAddress = "http://47.52.64.92:8081"
	config.Config.JenkinsToken = "Rancher123"
	config.Config.JenkinsUser = "root"
}

func TestSyncActivity(t *testing.T) {
	/*
		InitJenkins()
		var activity *pipeline.Activity
		actiStr := "{\"actions\":{},\"activity_stages\":[{\"activity_steps\":[{\"name\":\"step1_1\",\"status\":\"Waitting\"}],\"name\":\"stage1\",\"status\":\"Waiting\"},{\"activity_steps\":[{\"name\":\"step2_1\",\"status\":\"Waitting\"},{\"name\":\"step2_2\",\"status\":\"Waitting\"}],\"name\":\"stage3\",\"status\":\"Waiting\"}],\"id\":\"d6988c9f-b583-4be0-8aa2-8b5eab0d6ded\",\"links\":{\"pipeline\":\"http://localhost/v1/pipelines/:\",\"self\":\"http://localhost/v1/activitys/d6988c9f-b583-4be0-8aa2-8b5eab0d6ded\"},\"pipeline\":{\"actions\":null,\"branch\":\"master\",\"file\":\"\",\"id\":\"5345ecb6-2895-465a-887f-6eec2094bb73\",\"lastRunId\":\"5d4b0d12-ee1f-4011-8209-3e96cb027be4\",\"lastRunStatus\":\"Waitting\",\"links\":null,\"name\":\"ppl_scm\",\"repository\":\"https://github.com/rancher/pipeline.git\",\"runCount\":1,\"stages\":[{\"name\":\"stage1\",\"steps\":[{\"branch\":\"master\",\"name\":\"step1_1\",\"type\":\"scm\"}]},{\"name\":\"stage3\",\"ordinal\":1,\"steps\":[{\"command\":\"echo test in stage2\",\"image\":\"busybox\",\"name\":\"step2_1\",\"type\":\"task\"},{\"command\":\"echo test2 in stage2\",\"image\":\"busybox\",\"name\":\"step2_2\",\"type\":\"task\"}]}],\"trigger\":{\"spec\":\"\",\"type\":\"\"}},\"runSequence\":2,\"start_ts\":1.500260827498e+12,\"status\":\"Waitting\",\"type\":\"activity\"}"
		json.Unmarshal([]byte(actiStr), activity)
		t.Logf("activity:%v", activity)
		jp := JenkinsProvider{}
		res, err := jp.SyncActivity(activity)
		t.Logf("res,err:%v,%v", res, err)
	*/
}

func TestCommandBuild(t *testing.T) {

	testCases := map[string]struct {
		step   *pipeline.Step
		result string
	}{
		"build-sourcecode-notpush": {
			step: &pipeline.Step{
				Type:        "build",
				SourceType:  "sc",
				TargetImage: "testimage:latest",
				PushFlag:    false,
			},
			result: "docker build --tag testimage:latest .;",
		},
		"build-sourcecode-push": {
			step: &pipeline.Step{
				Type:        "build",
				SourceType:  "sc",
				TargetImage: "testimage:latest",
				PushFlag:    true,
				RegUserName: "lawr",
				RegPassword: "1",
			},
			result: "docker build --tag testimage:latest .;docker login --username lawr --password 1 ;docker push testimage:latest;",
		},
		"build-sourcecode-push-reg": {
			step: &pipeline.Step{
				Type:        "build",
				SourceType:  "sc",
				TargetImage: "127.0.0.1:8000/namespace/testimage:latest",
				PushFlag:    true,
				RegUserName: "lawr",
				RegPassword: "1",
			},
			result: "docker build --tag 127.0.0.1:8000/namespace/testimage:latest .;docker login --username lawr --password 1 127.0.0.1:8000;docker push 127.0.0.1:8000/namespace/testimage:latest;",
		},
	}

	for name, test := range testCases {
		t.Log("Test case:", name)
		result := commandBuilder(test.step)
		if result != test.result {
			t.Errorf("in commandbuild test expected \"%v\" but got \"%v\"", test.result, result)
		}
	}
}
