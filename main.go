package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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

func loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		log.Printf("[%d] %s %s — %v",
			c.Writer.Status(),
			c.Request.Method,
			c.Request.URL.Path,
			duration,
		)
	}
}

func apiKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" || key != os.Getenv("API_KEY") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing API key"})
			return
		}
		c.Next()
	}
}

func getTodos(c *gin.Context) {
	c.JSON(http.StatusOK, todos)
}

func createTodo(c *gin.Context) {
	var input struct {
		Title string `json:"title" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	todo := Todo{
		ID:    nextID,
		Title: input.Title,
		Done:  false,
	}

	todos = append(todos, todo)
	nextID++

	c.JSON(http.StatusCreated, todo)
}

func updateTodo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var input struct {
		Title string `json:"title"`
		Done  *bool  `json:"done"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for i, todo := range todos {
		if todo.ID == id {
			if input.Title != "" {
				todos[i].Title = input.Title
			}
			if input.Done != nil {
				todos[i].Done = *input.Done
			}
			c.JSON(http.StatusOK, todos[i])
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
}

func deleteTodo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	for i, todo := range todos {
		if todo.ID == id {
			todos = append(todos[:i], todos[i+1:]...)
			c.JSON(http.StatusOK, gin.H{"message": "todo deleted"})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
}

func main() {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"pong": true})
	})

	v1 := r.Group("/api/v1")
	v1.Use(loggerMiddleware())
	v1.Use(apiKeyMiddleware())
	{
		v1.GET("/todos", getTodos)
		v1.POST("/todos", createTodo)
		v1.PUT("/todos/:id", updateTodo)
		v1.DELETE("/todos/:id", deleteTodo)
	}

	r.Run(":8080")
}
