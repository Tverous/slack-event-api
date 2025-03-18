package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Slack event request structure
type SlackEvent struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
}

func main() {
	router := gin.Default()

	// Slack Events API endpoint
	router.POST("/slack/events", func(c *gin.Context) {
		var event SlackEvent

		// Parse incoming JSON request
		if err := c.BindJSON(&event); err != nil {
			log.Println("Error parsing Slack event:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Handle Slack's URL verification challenge
		if event.Type == "url_verification" {
			log.Println("URL verification request received")
			c.JSON(http.StatusOK, gin.H{"challenge": event.Challenge})
			return
		}

		// Log the event (you can add more handling logic here)
		fmt.Println("Slack event received:", event)

		// Respond to Slack with HTTP 200
		c.Status(http.StatusOK)
	})

	// Start server on port 3000
	port := "3000"
	fmt.Println("Server is running on port", port)
	err := router.Run(":" + port)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

