package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// SlackEvent represents the incoming Slack event structure
type SlackEvent struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge,omitempty"`
	Event     struct {
		Type    string `json:"type"`
		User    string `json:"user"`
		Text    string `json:"text,omitempty"`
		Channel string `json:"channel"`
		Files   []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Mimetype  string `json:"mimetype"`
			Filetype  string `json:"filetype"`
			UrlPrivate string `json:"url_private"`
		} `json:"files,omitempty"`
	} `json:"event"`
}

var (
	messages []SlackEvent // Store incoming messages
	mutex    sync.Mutex   // Ensure safe concurrent access
)

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

		// Process message events
		if event.Event.Type == "message" && event.Event.User != "" {
			mutex.Lock()
			messages = append(messages, event) // Store the message
			mutex.Unlock()
			log.Println("Message stored:", event.Event.Text)

			// Check if a file was uploaded
			if len(event.Event.Files) > 0 {
				for _, file := range event.Event.Files {
					log.Printf("File uploaded: Name=%s, Type=%s, URL=%s\n", file.Name, file.Filetype, file.UrlPrivate)
				}
			}
		}

		// Respond to Slack
		c.Status(http.StatusOK)
	})

	// New endpoint to retrieve stored messages
	router.GET("/messages", func(c *gin.Context) {
		mutex.Lock()
		defer mutex.Unlock()

		// Return stored messages as JSON
		c.JSON(http.StatusOK, messages)

		// Clear messages after retrieval to prevent duplicate processing
		messages = nil
	})

	// Start server on port 3000
	port := "3000"
	fmt.Println("Server is running on port", port)
	err := router.Run(":" + port)
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

