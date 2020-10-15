package main

import (
	"encoding/json"
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

func main() {
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

	if err := router.Run(fmt.Sprintf(":%s", config.PORT)); err != nil {
		log.Fatalf("Unable to start server %v", err)
	}
}
