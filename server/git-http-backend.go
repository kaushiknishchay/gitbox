/*
(The MIT License)

Copyright (c) 2013 Asim Aslam <asim@aslam.me>

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
'Software'), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
/**
This file source code is copied from below repo with slight modifications to work
with gin framework
https://github.com/asim/git-http-backend

All credits goes to the original author
*/
package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"git-on-web/config"
	"git-on-web/hub"
	"git-on-web/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Service struct {
	Method  string
	Handler func(HandlerReq)
	Rpc     string
}

type Config struct {
	AuthPassEnvVar string
	AuthUserEnvVar string
	DefaultEnv     string
	ProjectRoot    string
	GitBinPath     string
	UploadPack     bool
	ReceivePack    bool
	RoutePrefix    string
}

type HandlerReq struct {
	w        http.ResponseWriter
	r        *http.Request
	Rpc      string
	Dir      string
	File     string
	RepoName string
}

var (
	DefaultConfig = Config{
		AuthPassEnvVar: "",
		AuthUserEnvVar: "",
		DefaultEnv:     "",
		ProjectRoot:    config.REPO_BASE_DIR,
		GitBinPath:     "/usr/bin/git",
		UploadPack:     true,
		ReceivePack:    true,
		RoutePrefix:    "",
	}
)

var services = map[string]Service{
	"(.*?)/git-upload-pack$":                       Service{"POST", serviceRpc, "upload-pack"},
	"(.*?)/git-receive-pack$":                      Service{"POST", serviceRpc, "receive-pack"},
	"(.*?)/info/refs$":                             Service{"GET", getInfoRefs, ""},
	"(.*?)/HEAD$":                                  Service{"GET", getTextFile, ""},
	"(.*?)/objects/info/alternates$":               Service{"GET", getTextFile, ""},
	"(.*?)/objects/info/http-alternates$":          Service{"GET", getTextFile, ""},
	"(.*?)/objects/info/packs$":                    Service{"GET", getInfoPacks, ""},
	"(.*?)/objects/info/[^/]*$":                    Service{"GET", getTextFile, ""},
	"(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$":      Service{"GET", getLooseObject, ""},
	"(.*?)/objects/pack/pack-[0-9a-f]{40}\\.pack$": Service{"GET", getPackFile, ""},
	"(.*?)/objects/pack/pack-[0-9a-f]{40}\\.idx$":  Service{"GET", getIdxFile, ""},
}

var createSep = []byte{48, 48, 48, 48, 80, 65, 67, 75, 0, 0, 0, 2, 0, 0, 0}
var deleteSep = []byte{48, 48, 48, 48}

//GitOpsHandler handles git operations
func GitOpsHandler(c *gin.Context) {
	var w = c.Writer
	var r = c.Request

	repoName := c.Params.ByName("repo")

	for match, service := range services {
		re, err := regexp.Compile(match)
		if err != nil {
			log.Print(err)
		}

		if m := re.FindStringSubmatch(r.URL.Path); m != nil {

			if service.Method != r.Method {
				renderMethodNotAllowed(w, r)
				return
			}

			rpc := service.Rpc
			file := strings.Replace(r.URL.Path, m[1]+"/", "", 1)
			repoAbsolutePath := utils.GetRepoAbsolutePath(repoName)

			hr := HandlerReq{w, r, rpc, repoAbsolutePath, file, repoName}
			service.Handler(hr)
			return
		}
	}
	renderNotFound(w)
	return
}

func serviceRpc(hr HandlerReq) {
	w, r, rpc, dir, repoName := hr.w, hr.r, hr.Rpc, hr.Dir, hr.RepoName
	access := hasAccess(r, dir, rpc, true)

	if access == false {
		renderNoAccess(w)
		return
	}

	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	data, err := ioutil.ReadAll(r.Body)
	bodyReader := ioutil.NopCloser(bytes.NewReader(data))

	//check if the push is for creating/updating new ref
	nullByteIndex := bytes.Index(data, createSep)

	if nullByteIndex < 0 {
		//check if push is for deleting ref
		nullByteIndex = bytes.LastIndex(data, deleteSep)
	}

	if nullByteIndex > -1 {
		pushMetadata := data[:nullByteIndex]
		metaInfo, err := utils.ParseGitPushMetadata(pushMetadata)

		if err != nil {
			log.Printf("%v\n", err)
		} else {
			hub.SuperHubInstance.SendEventToRepo(repoName, metaInfo.Bytes())
		}
	}

	env := os.Environ()

	if DefaultConfig.DefaultEnv != "" {
		env = append(env, DefaultConfig.DefaultEnv)
	}

	user, password, authok := r.BasicAuth()
	if authok {
		if DefaultConfig.AuthUserEnvVar != "" {
			env = append(env, fmt.Sprintf("%s=%s", DefaultConfig.AuthUserEnvVar, user))
		}
		if DefaultConfig.AuthPassEnvVar != "" {
			env = append(env, fmt.Sprintf("%s=%s", DefaultConfig.AuthPassEnvVar, password))
		}
	}

	args := []string{rpc, "--stateless-rpc", dir}
	cmd := exec.Command(DefaultConfig.GitBinPath, args...)
	version := r.Header.Get("Git-Protocol")
	if len(version) != 0 {
		cmd.Env = append(env, fmt.Sprintf("GIT_PROTOCOL=%s", version))
	}
	cmd.Dir = dir
	cmd.Env = env
	in, err := cmd.StdinPipe()
	if err != nil {
		log.Print(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Print(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Print(err)
	}

	var reader io.ReadCloser
	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(bodyReader)
		defer reader.Close()
	default:
		reader = bodyReader
	}
	io.Copy(in, reader)
	in.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("expected http.ResponseWriter to be an http.Flusher")
	}

	p := make([]byte, 1024)
	for {
		n_read, err := stdout.Read(p)
		if err == io.EOF {
			break
		}
		n_write, err := w.Write(p[:n_read])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if n_read != n_write {
			fmt.Printf("failed to write data: %d read, %d written\n", n_read, n_write)
			os.Exit(1)
		}
		flusher.Flush()
	}

	cmd.Wait()
}

func getInfoRefs(hr HandlerReq) {
	w, r, dir := hr.w, hr.r, hr.Dir
	service_name := getServiceType(r)
	access := hasAccess(r, dir, service_name, false)
	version := r.Header.Get("Git-Protocol")
	if access {
		args := []string{service_name, "--stateless-rpc", "--advertise-refs", "."}
		refs := gitCommand(dir, version, args...)

		hdrNocache(w)
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", service_name))
		w.WriteHeader(http.StatusOK)
		if len(version) == 0 {
			w.Write(packetWrite("# service=git-" + service_name + "\n"))
			w.Write(packetFlush())
		}
		w.Write(refs)
	} else {
		updateServerInfo(dir)
		hdrNocache(w)
		sendFile("text/plain; charset=utf-8", hr)
	}
}

func getInfoPacks(hr HandlerReq) {
	hdrCacheForever(hr.w)
	sendFile("text/plain; charset=utf-8", hr)
}

func getLooseObject(hr HandlerReq) {
	hdrCacheForever(hr.w)
	sendFile("application/x-git-loose-object", hr)
}

func getPackFile(hr HandlerReq) {
	hdrCacheForever(hr.w)
	sendFile("application/x-git-packed-objects", hr)
}

func getIdxFile(hr HandlerReq) {
	hdrCacheForever(hr.w)
	sendFile("application/x-git-packed-objects-toc", hr)
}

func getTextFile(hr HandlerReq) {
	hdrNocache(hr.w)
	sendFile("text/plain", hr)
}

// Logic helping functions

func sendFile(content_type string, hr HandlerReq) {
	w, r := hr.w, hr.r
	req_file := path.Join(hr.Dir, hr.File)

	f, err := os.Stat(req_file)
	if os.IsNotExist(err) {
		renderNotFound(w)
		return
	}

	w.Header().Set("Content-Type", content_type)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", f.Size()))
	w.Header().Set("Last-Modified", f.ModTime().Format(http.TimeFormat))
	http.ServeFile(w, r, req_file)
}

func getServiceType(r *http.Request) string {
	serviceType := r.FormValue("service")

	if s := strings.HasPrefix(serviceType, "git-"); !s {
		return ""
	}

	return strings.Replace(serviceType, "git-", "", 1)
}

func hasAccess(r *http.Request, dir string, rpc string, check_content_type bool) bool {
	if check_content_type {
		if r.Header.Get("Content-Type") != fmt.Sprintf("application/x-git-%s-request", rpc) {
			return false
		}
	}

	if !(rpc == "upload-pack" || rpc == "receive-pack") {
		return false
	}
	if rpc == "receive-pack" {
		return DefaultConfig.ReceivePack
	}
	if rpc == "upload-pack" {
		return DefaultConfig.UploadPack
	}

	return getConfigSetting(rpc, dir)
}

func getConfigSetting(service_name string, dir string) bool {
	service_name = strings.Replace(service_name, "-", "", -1)
	setting := getGitConfig("http."+service_name, dir)

	if service_name == "uploadpack" {
		return setting != "false"
	}

	return setting == "true"
}

func getGitConfig(config_name string, dir string) string {
	args := []string{"config", config_name}
	out := string(gitCommand(dir, "", args...))
	return out[0 : len(out)-1]
}

func updateServerInfo(dir string) []byte {
	args := []string{"update-server-info"}
	return gitCommand(dir, "", args...)
}

func gitCommand(dir string, version string, args ...string) []byte {
	command := exec.Command(DefaultConfig.GitBinPath, args...)
	if len(version) > 0 {
		command.Env = append(os.Environ(), fmt.Sprintf("GIT_PROTOCOL=%s", version))
	}
	command.Dir = dir
	out, err := command.Output()

	if err != nil {
		log.Print(err)
	}

	return out
}

// HTTP error response handling functions

func renderMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Proto == "HTTP/1.1" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method Not Allowed"))
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}
}

func renderNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 Not Found"))
}

func renderNoAccess(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Forbidden"))
}

// Packet-line handling function

func packetFlush() []byte {
	return []byte("0000")
}

func packetWrite(str string) []byte {
	s := strconv.FormatInt(int64(len(str)+4), 16)

	if len(s)%4 != 0 {
		s = strings.Repeat("0", 4-len(s)%4) + s
	}

	return []byte(s + str)
}

// Header writing functions

func hdrNocache(w http.ResponseWriter) {
	w.Header().Set("Expires", "Fri, 01 Jan 1980 00:00:00 GMT")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
}

func hdrCacheForever(w http.ResponseWriter) {
	now := time.Now().Unix()
	expires := now + 31536000
	w.Header().Set("Date", fmt.Sprintf("%d", now))
	w.Header().Set("Expires", fmt.Sprintf("%d", expires))
	w.Header().Set("Cache-Control", "public, max-age=31536000")
}
