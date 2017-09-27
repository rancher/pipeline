package restfulserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/pipeline/config"
	"github.com/robfig/cron"
)

func testGetCurrentUser(t *testing.T) {
	config.Config.CattleUrl = "http://47.52.64.92:8080/v1"
	config.Config.CattleSecretKey = "DLgVx2M2YbQjC5cJ1EEugA7zU7M28tfFpH6Ft1ui"
	config.Config.CattleAccessKey = "64EE69CCFD50B38B9B20"
}

func TestCompuTimestamp(t *testing.T) {
	raw := "00h00m00s001ms  Started by user admin\n00h00m00s001ms  Building in workspace /var/jenkins_home/workspace/1a9ba6e6-74d0-4e92-b12e-688a4ceb2a8d\n00h00m00s020ms  Cloning the remote Git repository\n00h00m00s020ms  Cloning repository https://github.com/gitlawr/php.git\n00h00m00s020ms   > git init /var/jenkins_home/workspace/1a9ba6e6-74d0-4e92-b12e-688a4ceb2a8d # timeout=10\n00h00m00s028ms  Fetching upstream changes from https://github.com/gitlawr/php.git\n00h00m00s028ms   > git --version # timeout=10\n00h00m00s035ms   > git fetch --tags --progress https://github.com/gitlawr/php.git +refs/heads/*:refs/remotes/origin/*\n00h00m01s994ms   > git config remote.origin.url https://github.com/gitlawr/php.git # timeout=10\n00h00m02s010ms   > git config --add remote.origin.fetch +refs/heads/*:refs/remotes/origin/* # timeout=10\n00h00m02s019ms   > git config remote.origin.url https://github.com/gitlawr/php.git # timeout=10\n00h00m02s031ms  Fetching upstream changes from https://github.com/gitlawr/php.git\n00h00m02s031ms   > git fetch --tags --progress https://github.com/gitlawr/php.git +refs/heads/*:refs/remotes/origin/*\n00h00m03s412ms   > git rev-parse origin/master^{commit} # timeout=10\n00h00m03s419ms  Checking out Revision b79ab812941d0c87931cd2f3dd668b77d8c8970c (origin/master)\n00h00m03s420ms  Commit message: \"add docker file\"\n00h00m03s420ms   > git config core.sparsecheckout # timeout=10\n00h00m03s426ms   > git checkout -f b79ab812941d0c87931cd2f3dd668b77d8c8970c\n00h00m03s432ms  First time build. Skipping changelog.\n00h00m03s433ms  [1a9ba6e6-74d0-4e92-b12e-688a4ceb2a8d] $ /bin/sh -xe /tmp/jenkins4933243017033701954.sh\n00h00m03s441ms  Warning: you have no plugins providing access control for builds, so falling back to legacy behavior of permitting any downstream builds to be triggered\n00h00m03s441ms  Triggering a new build of basic_test_1a9ba6e6-74d0-4e92-b12e-688a4ceb2a8d\n00h00m03s441ms  Finished: SUCCESS"
	str, err := computeLogTimestamp(1501561615405, raw)
	t.Logf("get ts:\n%v\nerr:%v", str, err)

	raw = "00h00m00s001ms  Started by user admin\n00h00m00s001ms  Building in workspace /var/jenkins_home/workspace/1a9ba6e6-74d0-4e92-b12e-688a4ceb2a8d\n00h00m00s020ms  Cloning the remote Git repository\n00h00m00s020ms  Cloning repository https://github.com/gitlawr/php.git\n00h00m00s020ms   > git init /var/jenkins_home/workspace/1a9ba6e6-74d0-4e92-b12e-688a4ceb2a8d # timeout=10\n00h00m00s028ms  Fetching upstream changes from https://github.com/gitlawr/php.git\n00h00m00s028ms   > git --version # timeout=10\n00h00m00s035ms   > git fetch --tags --progress https://github.com/gitlawr/php.git +refs/heads/*:refs/remotes/origin/*\n00h00m01s994ms   > git config remote.origin.url https://github.com/gitlawr/php.git # timeout=10\n00h00m02s010ms   > git config --add remote.origin.fetch +refs/heads/*:refs/remotes/origin/* # timeout=10\n00h00m02s019ms   > git config remote.origin.url https://github.com/gitlawr/php.git # timeout=10\n00h00m02s031ms  Fetching upstream changes from https://github.com/gitlawr/php.git\n00h00m02s031ms   > git fetch --tags --progress https://github.com/gitlawr/php.git +refs/heads/*:refs/remotes/origin/*\n00h00m03s412ms   > git rev-parse origin/master^{commit} # timeout=10\n00h00m03s419ms  Checking out Revision b79ab812941d0c87931cd2f3dd668b77d8c8970c (origin/master)\n00h00m03s420ms  Commit message: \"add docker file\"\n00h00m03s420ms   > git config core.sparsecheckout # timeout=10\n00h00m03s426ms   > git checkout -f b79ab812941d0c87931cd2f3dd668b77d8c8970c\n00h00m03s432ms  First time build. Skipping changelog.\n00h00m03s433ms  [1a9ba6e6-74d0-4e92-b12e-688a4ceb2a8d] $ /bin/sh -xe /tmp/jenkins4933243017033701954.sh\n00h00m03s441ms  Warning: you have no plugins providing access control for builds, so falling back to legacy behavior of permitting any downstream builds to be triggered\n00h00m03s441ms  Triggering a new build of basic_test_1a9ba6e6-74d0-4e92-b12e-688a4ceb2a8d\n  Finished: SUCCESS"
	str, err = computeLogTimestamp(1501561615405, raw)
	t.Logf("get ts:\n%v\nerr:%v", str, err)

}

func TestGetUser(t *testing.T) {
	token := "45d35661fa5fc4e03ab8959d6290a1a2819eabe3"
	_, err := getGithubUser(token)
	if err != nil {
		fmt.Print(err)
	}
}

func TestUnmarshal(t *testing.T) {
	var obj interface{}
	b := bytes.NewBufferString("{\"abc\":1}").Bytes()
	logrus.Print(b)
	json.Unmarshal(b, obj)
	logrus.Print(b)
	a := foo(b)
	logrus.Print(a)
	c := a.([]byte)
	logrus.Print(c)

	u, err := url.Parse("http://bing.com/search?q=dotnet")
	if err != nil {
		log.Fatal(err)
	}
	u.Scheme = "https"
	u.Host = "google.com"
	q := u.Query()
	q.Set("qq", "golang")
	u.RawQuery = q.Encode()
	fmt.Println(u)
}

func foo(obj interface{}) interface{} {
	return obj
}
func TestTimeParse(t *testing.T) {
	nextRunTime := int64(0)
	spec := "30 * * * *"

	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		logrus.Errorf("error parse cron exp,%v,%v", spec, err)
	}
	nextRunTime = schedule.Next(time.Now()).UnixNano() / int64(time.Millisecond)
	t.Logf("spec:%v,nextRun:%v", spec, nextRunTime)
}
