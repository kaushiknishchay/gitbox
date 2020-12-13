package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "gitbox"
	"gitbox/utils"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_RepoCreateEndpoint(t *testing.T) {
	ts := httptest.NewServer(SetupServer())
	defer ts.Close()

	generateRepoName := fmt.Sprintf("repo-%v", time.Now().Unix())

	requestBody, err := json.Marshal(map[string]string{
		"name": generateRepoName,
	})

	if err != nil {
		t.Fatalf("error, %v", err)
	}

	response, err := http.Post(fmt.Sprintf("%s/repo", ts.URL), "application/json", bytes.NewBuffer(requestBody))

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		t.Fatalf("Expected status code 200, got %v", response.StatusCode)
	}

	var bodyJSON RepoCreateResponse

	bodyBuffer, err := ioutil.ReadAll(response.Body)

	if err != nil {
		t.Fatalf("error, %v", err)
	}

	err = json.Unmarshal(bodyBuffer, &bodyJSON)

	if err != nil {
		t.Fatalf("Body read error : %v", err)
	}

	if !bodyJSON.Status {
		t.Fatalf("Expected response status to be true")
	}

	if bodyJSON.RepoName != generateRepoName {
		t.Fatalf("Expected create repo name to match the sent repo name")
	}

	repoAbsolutePath := utils.GetRepoAbsolutePath(generateRepoName)
	_ = utils.RemoveRepoAtPath(repoAbsolutePath)
}
