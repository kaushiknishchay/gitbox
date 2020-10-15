package utils

import (
	"bytes"
	"errors"
	"git-on-web/config"
	"log"
	"os"
	"os/exec"
	"path"
)

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
		if err := os.RemoveAll(repoAbsolutePath); err != nil {
			return err
		}
		return err
	}

	return nil
}
func GetCommitsLog(repoName string) (string, error) {
	repoAbsolutePath := GetRepoAbsolutePath(repoName)

	logCommand := exec.Command("git", "log", `--pretty=format:{ "commit": "%H", "subject": "%s", "body": "%b", "author": { "name": "%aN", "email": "%aE", "date": "%aD" }, "commiter": { "name": "%cN", "email": "%cE", "date": "%cD" }%n},`)

	logCommand.Dir = repoAbsolutePath
	logStdoutBuffer, err := logCommand.StdoutPipe()

	if err != nil {
		log.Print(err)
		return "", err
	}

	if err := logCommand.Start(); err != nil {
		log.Print(err)
		return "", err
	}

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(logStdoutBuffer)
	logsOutput := buf.String()

	if err := logCommand.Wait(); err != nil {
		log.Print(err)
		return "", err
	}

	return logsOutput, nil
}
