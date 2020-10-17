package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"git-on-web/config"
	"git-on-web/server"
	"git-on-web/utils"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"strings"
)

type RepoCreateRequest struct {
	RepoName string `json:"name" binding:"required"`
}

type RepoCreateResponse struct {
	Status   bool   `json:"status"`
	RepoName string `json:"repoName"`
}

func addGitRoutes(gitOps *gin.RouterGroup) {
	gitOps.Use(func(c *gin.Context) {
		repoName := c.Params.ByName("repo")
		exists := utils.CheckRepoExists(repoName)

		if exists == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"status": false,
				"error":  "repo not found",
			})
			c.Abort()
			return
		}

		c.Next()
	})

	gitOps.Any("/*action", func(c *gin.Context) {
		action := c.Param("action")

		switch action {
		case "/":
			c.JSON(http.StatusOK, gin.H{
				"status": true,
			})
		case "/log":
			repoName := c.Params.ByName("repo")

			logsObject, err := utils.GetCommitsLog(repoName)
			logsArrayString := fmt.Sprintf("[%s]", strings.TrimSuffix(logsObject, ","))

			var logsJSON []interface{}

			if err := json.Unmarshal([]byte(logsArrayString), &logsJSON); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": false,
					"error":  "unable to output logs",
				})
				return
			}

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": false,
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"status": true,
				"logs":   logsJSON,
			})

		default:
			server.GitOpsHandler(c)
		}
	})
}

func setupServer() *gin.Engine {

	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello golang gin!")
	})

	router.POST("/repo", func(c *gin.Context) {
		var request RepoCreateRequest

		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   err.Error(),
				"message": "Only JSON requests are allowed",
			})
			return
		}

		if err := utils.CheckRepoExists(request.RepoName); err != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": err.Error(),
			})
			return
		}

		if err := utils.CreateNewRepo(request.RepoName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   true,
			"repoName": request.RepoName,
		})
	})

	gitOps := router.Group("/git/:repo")
	addGitRoutes(gitOps)

	return router
}

func main() {
	flag.StringVar(&config.PORT, "port", "9090", "port on which to run the server. Default: 9090")
	flag.StringVar(&config.REPO_BASE_DIR, "repos", "/tmp/repos", "directory where repos will be created. Default: /tmp/repos")
	flag.Parse()

	ginRouter := setupServer()

	if err := ginRouter.Run(fmt.Sprintf(":%s", config.PORT)); err != nil {
		log.Fatalf("Unable to start server %v", err)
	}
}
