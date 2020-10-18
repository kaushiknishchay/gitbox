package main

import (
	"flag"
	"fmt"
	"git-on-web/config"
	"git-on-web/server"
	"git-on-web/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
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

		if repoNameValid := utils.IsRepoNameValid(repoName); !repoNameValid {
			c.JSON(http.StatusNotFound, gin.H{
				"status": false,
				"error":  "invalid repo name",
			})
			c.Abort()
			return
		}

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

			logsJSON, err := utils.GetCommitsLog(repoName)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status": false,
					"error":  err.Error(),
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

		if repoNameValid := utils.IsRepoNameValid(request.RepoName); !repoNameValid {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "repo name can only contain alpha numeric characters and '-' or '_'",
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
