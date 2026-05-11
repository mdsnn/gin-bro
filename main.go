package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("database unreachable:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id    SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			done  BOOLEAN NOT NULL DEFAULT FALSE
		)
	`)
	if err != nil {
		log.Fatal("failed to create table:", err)
	}

	log.Println("database connected and ready")
}

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
	rows, err := db.Query("SELECT id, title, done FROM todos ORDER BY id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	result := []Todo{}
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Done); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		result = append(result, t)
	}

	c.JSON(http.StatusOK, result)
}

func createTodo(c *gin.Context) {
	var input struct {
		Title string `json:"title" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var todo Todo
	err := db.QueryRow(
		"INSERT INTO todos (title, done) VALUES ($1, $2) RETURNING id, title, done",
		input.Title, false,
	).Scan(&todo.ID, &todo.Title, &todo.Done)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	var todo Todo
	err = db.QueryRow("SELECT id, title, done FROM todos WHERE id = $1", id).
		Scan(&todo.ID, &todo.Title, &todo.Done)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if input.Title != "" {
		todo.Title = input.Title
	}
	if input.Done != nil {
		todo.Done = *input.Done
	}

	err = db.QueryRow(
		"UPDATE todos SET title = $1, done = $2 WHERE id = $3 RETURNING id, title, done",
		todo.Title, todo.Done, id,
	).Scan(&todo.ID, &todo.Title, &todo.Done)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, todo)
}

func deleteTodo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	result, err := db.Exec("DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "todo not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "todo deleted"})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}

	initDB()
	defer db.Close()

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
