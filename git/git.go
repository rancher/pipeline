package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

func Clone(path, url, branch string) error {
	return runcmd("git", "clone", "-b", branch, "--single-branch", url, path)
}

func BranchHeadCommit(url, branch string) (string, error) {
	cmd := exec.Command("git", "ls-remote", url, branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrap(err, string(output))
	}
	strs := strings.Split(string(output), "\t")
	return strs[0], nil
}

func Init(path string, url string) error {
	return runcmd("git", "init", path)
}

func Push(path, repo, refspec string) error {
	return runcmd("git", "-C", path, "push", repo, refspec)
}

//add,commit and push changes.
func LazyPush(path, repo, refspec string) error {
	err := runcmd("git", "-C", path, "add", ".")
	if err != nil {
		return err
	}

	err = runcmd("git", "-C", path, "commit", "-m", "updating")
	if err != nil {
		return err
	}
	//-f is added for test purpose, remove later.
	err = runcmd("git", "-C", path, "push", repo, refspec, "-f")
	return err
}
func Update(path, branch string) error {
	if err := runcmd("git", "-C", path, "fetch"); err != nil {
		return err
	}
	return runcmd("git", "-C", path, "checkout", fmt.Sprintf("origin/%s", branch))
}

func HeadCommit(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "HEAD")
	output, err := cmd.Output()
	return strings.Trim(string(output), "\n"), err
}

func IsValid(url string) bool {
	err := runcmd("git", "ls-remote", url)
	return (err == nil)
}

func runcmd(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	if log.GetLevel() >= log.DebugLevel {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return cmd.Run()
}
