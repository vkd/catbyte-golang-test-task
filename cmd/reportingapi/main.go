package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// TODO: configuration
const (
	redisAddr = "localhost:6379"
)

type MessageRequest struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Message  string `json:"message"`
}

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// TODO: release mode
	// TODO: custom middlewares
	r := gin.Default()

	r.GET("/message/list", func(c *gin.Context) {
		sender := c.Query("sender")
		receiver := c.Query("receiver")

		// TODO: check if empty

		res, err := rdb.XRange(c, fmt.Sprintf("%s|%s", sender, receiver), "-", "+").Result()
		if err != nil {
			log.Printf("Error on XREAD: %v", err)
			// TODO: error format
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "internal_error",
				"error":  err.Error(), // TODO: hide error details
			})
			return
		}
		log.Printf("stream is read")

		messages := make([]string, 0, len(res))
		for _, m := range res {
			// TODO: check if r.Stream is correct
			message, ok := m.Values["message"].(string)
			if !ok {
				log.Printf("Unknown message type: %T", m.Values["message"])
				continue
			}
			messages = append(messages, message)
		}

		c.JSON(200, gin.H{
			"sender":   sender,
			"receiver": receiver,
			"messages": messages,
		})
	})

	// TODO: configurable address/port
	r.Run("localhost:9001")
}
