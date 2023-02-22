package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// TODO: configuration
var (
	amqpURL       = GetEnv("AMQP_URL", "amqp://user:password@localhost:7001/")
	amqpQueueName = "events"

	redisAddr = GetEnv("REDIS_ADDR", "localhost:6379")
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
	// RabbitMQ
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

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// err = rdb.Set(ctx, "key", "value", 0).Err()
	// if err != nil {
	// 	panic(err)
	// }

	// val, err := rdb.Get(ctx, "key").Result()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("key", val)

	// val2, err := rdb.Get(ctx, "key2").Result()
	// if err == redis.Nil {
	// 	fmt.Println("key2 does not exist")
	// } else if err != nil {
	// 	panic(err)
	// } else {
	// 	fmt.Println("key2", val2)
	// }
	// Output: key value
	// key2 does not exist

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			// TODO: gracefully stop
			log.Printf("Signal SIGINT or SIGKILL is received")
			log.Printf("Shutting down...")
			return
		case d := <-msgs:
			var req MessageRequest
			err := json.Unmarshal(d.Body, &req)
			if err != nil {
				log.Printf("Error on unmarshal request body: %v", err)
				// TODO: d.Acknowledger.Nack() or requeque if error
				continue
			}

			log.Printf("Got a message: %+v", req)

			err = rdb.XAdd(ctx, &redis.XAddArgs{
				// TODO: filter our '|' char from sender/receiver names
				Stream: fmt.Sprintf("%s|%s", req.Sender, req.Receiver),
				Values: []string{"message", req.Message},
			}).Err()
			if err != nil {
				log.Printf("Error on write message into redis: %v", err)
				// TODO: d.Acknowledger.Nack() or requeque if error
				continue
			}

			log.Printf("Message is processed")
		}
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
