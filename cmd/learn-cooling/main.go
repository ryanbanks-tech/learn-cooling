package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	// Serve static files from /web/build
	router.Static("/static", "./web/build/static")

	// Serve index.html from /web/build
	router.GET("/", func(c *gin.Context) {
		c.File("./web/build/index.html")
	})

	log.Println("Server started at :8080")
	router.Run(":8080")
}
