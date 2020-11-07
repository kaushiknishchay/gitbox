package utils

import (
	"golang-app/config"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func generateRandomRepoName(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

func BenchmarkGetCommitsLogOnTestRepo(b *testing.B) {
	a := "/tmp/repos"
	config.REPO_BASE_DIR = a

	for n := 0; n < b.N; n++ {
		_, _ = GetCommitsLog("test-repo", 0)
	}
}

func BenchmarkGetRepoAbsolutePath(b *testing.B) {
	for n := 0; n < b.N; n++ {
		IsRepoNameValid(generateRandomRepoName(rand.Intn(30)))
	}
}

func TestGetCommitsLog(t *testing.T) {
	config.REPO_BASE_DIR = "/tmp/repos"

	type args struct {
		repoName string
	}

	tests := []struct {
		name    string
		args    args
		want    []CommitItem
		wantErr bool
	}{
		{"1", args{"test"}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCommitsLog(tt.args.repoName, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommitsLog() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCommitsLog() = %v, want %v", got, tt.want)
			}
		})
	}
}
