package main

import (
	"flag"
	"fmt"
	"golang-app/config"
	"golang-app/hub"
	"golang-app/server"
	"golang-app/utils"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// RepoCreateRequest structure of request
type RepoCreateRequest struct {
	RepoName string `json:"name" binding:"required"`
}

// RepoCreateResponse structure of response
type RepoCreateResponse struct {
	Status   bool   `json:"status"`
	RepoName string `json:"repoName"`
}

// addGitRoutes Setup all git operation related routes
//nolint:funlen
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
			pageNum, err := strconv.ParseInt(c.DefaultQuery("page", "0"), 10, 32)

			if err != nil || pageNum < 0 {
				pageNum = 0
			}

			logsJSON, err := utils.GetCommitsLog(repoName, pageNum)

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

func addWebSocketRoutes(webSockets *gin.RouterGroup) {
	webSockets.Any("/*action", func(c *gin.Context) {
		repoName := c.Params.ByName("repo")
		_ = c.Param("action")

		if _, ok := hub.SuperHubInstance[repoName]; !ok {
			hub.SuperHubInstance[repoName] = hub.CreateNewHub(repoName)
			go hub.SuperHubInstance[repoName].Run()
		}

		var upgrader websocket.Upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		connection, err := upgrader.Upgrade(c.Writer, c.Request, nil)

		if err != nil {
			log.Printf("%v", err.Error())
			return
		}

		client := &hub.Client{
			Hub:  hub.SuperHubInstance[repoName],
			Conn: connection,
			Send: make(chan []byte, 256),
		}

		client.Hub.Register <- client

		// start a go routine which will send messages to this client when something comes on channel
		go client.Write()
	})
}

func SetupServer() *gin.Engine {
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

	webSockets := router.Group("/ws/:repo")
	addWebSocketRoutes(webSockets)

	return router
}

func main() {
	flag.StringVar(&config.PORT, "port", "9090", "port on which to run the server. Default: 9090")
	flag.StringVar(&config.REPO_BASE_DIR, "repos", "/tmp/repos", "directory where repos will be created. Default: /tmp/repos")
	flag.Parse()

	ginRouter := SetupServer()

	if err := ginRouter.Run(fmt.Sprintf(":%s", config.PORT)); err != nil {
		log.Fatalf("Unable to start server %v", err)
	}
}
