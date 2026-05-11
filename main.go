package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

var (
	todos  = []Todo{}
	nextID = 1
)

func getTodos(c *gin.Context) {
	c.JSON(http.StatusOK, todos)
}

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello, World!"})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})

	r.GET("/todos", getTodos)

	r.Run(":8080")
}
