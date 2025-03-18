package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

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
            ID         string `json:"id"`
            Name       string `json:"name"`
            Mimetype   string `json:"mimetype"`
            Filetype   string `json:"filetype"`
            UrlPrivate string `json:"url_private"`
        } `json:"files,omitempty"`
    } `json:"event"`
}

var (
    messages []SlackEvent // Store incoming messages
    mutex    sync.Mutex   // Ensure safe concurrent access
)

// Verify Slack request signature
func isValidSlackRequest(c *gin.Context) bool {
    signingSecret := os.Getenv("SLACK_SIGNING_SECRET") // Set this in Render
    if signingSecret == "" {
        log.Println("SLACK_SIGNING_SECRET is not set!")
        return false
    }

    timestamp := c.GetHeader("X-Slack-Request-Timestamp")
    if timestamp == "" {
        log.Println("Missing timestamp header")
        return false
    }

    // Check if the timestamp is too old (prevents replay attacks)
    ts, err := strconv.ParseInt(timestamp, 10, 64)
    if err != nil || time.Now().Unix()-ts > 300 {
        log.Println("Request timestamp is too old or invalid")
        return false
    }

    // Read request body
    body, err := ioutil.ReadAll(c.Request.Body)
    if err != nil {
        log.Println("Failed to read request body")
        return false
    }

    // Restore request body for Gin context (since ioutil.ReadAll drains it)
    c.Request.Body = ioutil.NopCloser(strings.NewReader(string(body)))

    // Reconstruct the signature base string
    baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))

    // Compute HMAC-SHA256 signature
    h := hmac.New(sha256.New, []byte(signingSecret))
    h.Write([]byte(baseString))
    computedSignature := "v0=" + hex.EncodeToString(h.Sum(nil))

    // Compare computed signature with Slack's signature
    slackSignature := c.GetHeader("X-Slack-Signature")
    if !hmac.Equal([]byte(computedSignature), []byte(slackSignature)) {
        log.Println("Slack signature verification failed")
        return false
    }

    return true
}

func main() {
    router := gin.Default()

    // Slack Events API endpoint
    router.POST("/", func(c *gin.Context) {
        // Verify Slack request
        if !isValidSlackRequest(c) {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized request"})
            return
        }

        var event SlackEvent
        if err := c.BindJSON(&event); err != nil {
            log.Printf("Error parsing Slack event: %v\n", err)
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
            return
        }

        // Handle Slack's URL verification challenge
        if event.Type == "url_verification" {
            log.Println("URL verification request received")
            c.JSON(http.StatusOK, gin.H{"challenge": event.Challenge})
            return
        }

        // Respond to Slack immediately to prevent retries
        c.Status(http.StatusOK)

        // Process the event in a separate goroutine
        go func() {
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
        }()
    })

    // Endpoint to retrieve stored messages
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
