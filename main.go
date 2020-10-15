package main

import (
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello golang gin!")
	})

	if err := r.Run(); err != nil {
		log.Fatalf("Unable to start server %v", err)
	}
}
