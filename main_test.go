package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"git-on-web/utils"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func Test_RepoCreateEndpoint(t *testing.T) {
	ts := httptest.NewServer(setupServer())
	defer ts.Close()

	generateRepoName := fmt.Sprintf("repo-%v", time.Now().Unix())

	requestBody, err := json.Marshal(map[string]string{
		"name": generateRepoName,
	})

	response, err := http.Post(fmt.Sprintf("%s/repo", ts.URL), "application/json", bytes.NewBuffer(requestBody))

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		t.Fatalf("Expected status code 200, got %v", response.StatusCode)
	}

	var bodyJson RepoCreateResponse

	bodyBuffer, err := ioutil.ReadAll(response.Body)
	err = json.Unmarshal(bodyBuffer, &bodyJson)

	if err != nil {
		t.Fatalf("Body read error : %v", err)
	}

	if !bodyJson.Status {
		t.Fatalf("Expected response status to be true")
	}

	if bodyJson.RepoName != generateRepoName {
		t.Fatalf("Expected create repo name to match the sent repo name")
	}

	repoAbsolutePath := utils.GetRepoAbsolutePath(generateRepoName)
	_ = utils.RemoveRepoAtPath(repoAbsolutePath)
}
