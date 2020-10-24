package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"git-on-web/config"
	"io"
	"log"
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
func GetCommitsLog(repoName string) ([]CommitItem, error) {
	repoAbsolutePath := GetRepoAbsolutePath(repoName)

	logCommand := exec.Command("git", "log", `--pretty=format:{ "commit": "%H", "subject": "%s", "body": "%b", "author": { "name": "%aN", "email": "%aE", "date": "%aD" }, "commiter": { "name": "%cN", "email": "%cE", "date": "%cD" }%n},`)

	logCommand.Dir = repoAbsolutePath
	logStdoutBuffer, err := logCommand.StdoutPipe()

	var logsJSON []CommitItem

	if err != nil {
		log.Print(err)
		return logsJSON, err
	}

	if err := logCommand.Start(); err != nil {
		log.Print(err)
		return logsJSON, err
	}

	buf := new(strings.Builder)
	_, err = io.Copy(buf, logStdoutBuffer)
	logsOutput := fmt.Sprintf("[%s]", strings.TrimSuffix(buf.String(), ","))

	if logsOutput == "[]" {
		return logsJSON, nil
	}

	if err := logCommand.Wait(); err != nil {
		log.Print(err)
		return logsJSON, err
	}

	if err := json.Unmarshal([]byte(logsOutput), &logsJSON); err != nil {
		return logsJSON, err
	}

	return logsJSON, nil
}
