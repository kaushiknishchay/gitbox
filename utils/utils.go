package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"git-on-web/config"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

type Author struct {
	Date  string `json:"date"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type CommiterType struct {
	Date  string `json:"date"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type CommitItem struct {
	Author   Author       `json:"author"`
	Body     string       `json:"body"`
	Commit   string       `json:"commit"`
	Commiter CommiterType `json:"commiter"`
	Subject  string       `json:"subject"`
}

var repoCheckRegEx = regexp.MustCompile(`^[a-zA-Z\-_0-9]+$`).MatchString

const perPageCount int64 = 100
const commitSeparator string = "^^$$^^$$"
const gitLogFormat string = `--pretty=format:{"commit": "%H","subject": "%s","body": "%b","author": {"name": "%aN", "email": "%aE", "date": "%ad"},"commiter": {"name": "%cN", "email": "%cE", "date": "%cd"}}` + commitSeparator

//IsRepoNameValid Checks if repo name is valid and contains only alphanumeric chars
func IsRepoNameValid(repoName string) bool {
	return repoCheckRegEx(repoName)
}

//GetRepoAbsolutePath Get absolute repo path in repo base dir
func GetRepoAbsolutePath(repoName string) string {
	return path.Join(config.REPO_BASE_DIR, repoName)
}

func CheckRepoExists(repoName string) error {
	repoAbsolutePath := path.Join(config.REPO_BASE_DIR, repoName)

	if _, err := os.Stat(repoAbsolutePath); os.IsNotExist(err) {
		return nil
	}

	return errors.New("repo with name already exists")
}

func RemoveRepoAtPath(repoAbsolutePath string) error {
	if err := os.RemoveAll(repoAbsolutePath); err != nil {
		return err
	}
	return nil
}

func CreateNewRepo(repoName string) error {
	repoAbsolutePath := GetRepoAbsolutePath(repoName)

	if err := os.MkdirAll(repoAbsolutePath, 0700); err != nil {
		return err
	}

	initCommand := exec.Command("git", "init", "--bare")
	// Change the directory where to run command
	initCommand.Dir = repoAbsolutePath

	if err := initCommand.Run(); err != nil {
		// remove directory if git init fails
		if err := RemoveRepoAtPath(repoAbsolutePath); err != nil {
			return err
		}
		return err
	}

	return nil
}

// GetCommitsLog to fetch commits log as json array
func GetCommitsLog(repoName string, pageNum int64) ([]CommitItem, error) {
	logCommand := exec.Command("git", "log", "--date=iso-strict", gitLogFormat, fmt.Sprintf("-n %d", perPageCount), fmt.Sprintf("--skip=%d", pageNum*perPageCount))

	logCommand.Dir = GetRepoAbsolutePath(repoName)
	out, _ := logCommand.Output()

	logOut := strings.Split(string(out), "^^$$^^$$")

	var gitCommitList []CommitItem
	var commitItem CommitItem
	for _, singleLog := range logOut {

		if singleLog == "" {
			continue
		}

		singleLog = strings.Replace(strings.TrimSpace(singleLog), "\n", `\n`, -1)

		err := json.Unmarshal([]byte(singleLog), &commitItem)
		if err != nil {
			continue
		}
		gitCommitList = append(gitCommitList, commitItem)
	}

	return gitCommitList, nil
}
