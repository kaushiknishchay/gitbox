package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"golang-app/config"
	"golang-app/models"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

// Author data present in single commit's Author field
type Author struct {
	Date  string `json:"date"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// CommiterType data present in single commit's commiter field
type CommiterType struct {
	Date  string `json:"date"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// CommitItem data present in single commit
type CommitItem struct {
	Author   Author       `json:"author"`
	Body     string       `json:"body"`
	Commit   string       `json:"commit"`
	Commiter CommiterType `json:"commiter"`
	Subject  string       `json:"subject"`
}

// CommitLogs all commits
type CommitLogs []CommitItem

var repoCheckRegEx = regexp.MustCompile(`^[a-zA-Z\-_0-9]+$`).MatchString

const nullSha string = "0000000000000000000000000000000000000000"

// IsRepoNameValid Checks if repo name is valid and contains only alphanumeric chars
func IsRepoNameValid(repoName string) bool {
	return repoCheckRegEx(repoName)
}

// GetRepoAbsolutePath Get absolute repo path in repo base dir
func GetRepoAbsolutePath(repoName string) string {
	return path.Join(config.REPO_BASE_DIR, repoName)
}

// CheckRepoExists Check is repo with name already present
func CheckRepoExists(repoName string) error {
	repoAbsolutePath := path.Join(config.REPO_BASE_DIR, repoName)

	if _, err := os.Stat(repoAbsolutePath); os.IsNotExist(err) {
		return nil
	}

	return errors.New("repo with name already exists")
}

// RemoveRepoAtPath remove directory at the path given
func RemoveRepoAtPath(repoAbsolutePath string) error {
	if err := os.RemoveAll(repoAbsolutePath); err != nil {
		return err
	}

	return nil
}

// CreateNewRepo initialize a bare git repo with given name
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
	pageArg := fmt.Sprintf("-n %d --skip=%d", config.PerPageCommitCount, pageNum*config.PerPageCommitCount)

	logCommand := exec.Command(
		"git",
		"log",
		"--date=iso-strict",
		config.GitLogFormat,
		pageArg,
	)

	logCommand.Dir = GetRepoAbsolutePath(repoName)
	out, _ := logCommand.Output()

	logOut := strings.Split(string(out), "^^$$^^$$")

	var gitCommitList []CommitItem

	var commitItem CommitItem

	for _, singleLog := range logOut {
		if singleLog == "" {
			continue
		}

		singleLog = strings.ReplaceAll(strings.TrimSpace(singleLog), "\n", `\n`)

		err := json.Unmarshal([]byte(singleLog), &commitItem)
		if err != nil {
			continue
		}

		gitCommitList = append(gitCommitList, commitItem)
	}

	return gitCommitList, nil
}

// ParseGitPushMetadata parse the metadat bytearray into a more informative struct
func ParseGitPushMetadata(metadata []byte) (*models.MetadataInfo, error) {
	// the starting 4 chars are not part of sha
	metadataPieces := bytes.Split(metadata[4:], []byte(" "))
	if len(metadataPieces) < 3 {
		return nil, errors.New("cannot parse git push info")
	}

	oldSha := string(metadataPieces[0])
	newSha := string(metadataPieces[1])
	// we have a null character after the ref, remove that
	ref := string(metadataPieces[2][:len(metadataPieces[2])-1])

	typeOfPush := ""

	switch {
	case oldSha == nullSha:
		typeOfPush = models.MetaInfoType_Create
	case newSha == nullSha:
		typeOfPush = models.MetaInfoType_Delete
	default:
		typeOfPush = models.MetaInfoType_Update
	}

	return &models.MetadataInfo{
		Ref:    ref,
		OldSha: oldSha,
		NewSha: newSha,
		Type:   typeOfPush,
	}, nil
}
