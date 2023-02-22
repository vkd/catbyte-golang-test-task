package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

// TODO: configuration
var (
	amqpURL       = GetEnv("AMQP_URL", "amqp://user:password@localhost:7001/")
	amqpQueueName = "events"
)

func GetEnv(k string, d string) string {
	v, ok := os.LookupEnv(k)
	if !ok {
		return d
	}
	return v
}

type MessageRequest struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Message  string `json:"message"`
}

func main() {
	// TODO: move to separate function
	conn, err := amqp.Dial(amqpURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		amqpQueueName, // name
		false,         // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// TODO: release mode
	// TODO: custom middlewares
	r := gin.Default()

	r.POST("/message", func(c *gin.Context) {
		var req MessageRequest
		err := c.BindJSON(&req)
		if err != nil {
			// TODO: error message format
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "bad_request",
				// TODO: hide detailed error message
				"error": err.Error(),
			})
			return
		}

		// TODO: investigate of better inner format of message
		bs, err := json.Marshal(req)
		if err != nil {
			// TODO: almost impossible
			c.JSON(http.StatusBadRequest, gin.H{
				"status": "bad_request",
				"error":  err.Error(),
			})
			return
		}

		err = ch.PublishWithContext(c,
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        bs,
			})
		if err != nil {
			// TODO: task requirements say "otherwise Bad Request"
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "internal_error",
				// TODO: hide internal details
				"error": err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// TODO: configurable address/port
	r.Run("localhost:9000")
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
