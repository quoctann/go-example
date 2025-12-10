package queue

import (
	"bytes"
	"context"
	"flag"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

/*
Experimental in local
docker run -it --rm --name rabbitmq -p 5672:5672 -p 15672:15672 rabbitmq:4-management

Server RabbitMQ: amqp://localhost:5672
WebUI: http://localhost:15672
Default only in localhost: guest/guest

A producer is a user application that sends messages.
A queue is a buffer that stores messages.
A consumer is a user application that receives messages.

Producer -> Exchange -[binding]-> Queue -> Consumer

Exchange types:
	direct:
	topic:
	headers:
	fanout: broadcast to all queues it knows

https://www.rabbitmq.com/tutorials/tutorial-one-go

go run main.go c >> start consumer
go run main.go >> start producer
*/

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// MARK: HELLO WORLD

// producer -> queue
func sendHelloWorld() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	// channel, which is where most of the API for getting things done resides
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// declaring a queue is idempotent - it will only be created if it doesn't exist already
	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	body := "Hello World!"
	err = ch.PublishWithContext(ctx,
		"",     // exchange - default by empty string
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s\n", body)
}

// queue -> consumer
func receiveHelloWorld() {
	// same as send declaration
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// start the consumer before the publisher, we want to make sure the queue
	// exists before we try to consume messages from it
	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// consume
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

	var forever chan struct{} = make(chan struct{})

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// run hello world example
func RunHelloWorldRMQ(skip bool) {
	if skip {
		return
	}

	if len(os.Args) < 2 {
		// publisher
		sendHelloWorld()
		return
	}

	flag.NewFlagSet("c", flag.ContinueOnError)
	if os.Args[1] != "c" {
		log.Println("Nothing run")
		return
	}

	// consumer
	receiveHelloWorld()
}

// MARK: WORK QUEUE

func newTask() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// q, err := ch.QueueDeclare(
	// 	"hello", // name
	// 	false,   // durable
	// 	false,   // delete when unused
	// 	false,   // exclusive
	// 	false,   // no-wait
	// 	nil,     // arguments
	// )

	q, err := ch.QueueDeclare(
		"task_queue", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Failed to declare a queue")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	//

	body := "Hello....."
	err = ch.PublishWithContext(ctx,
		"",     // exchange - default by empty string
		q.Name, // routing key
		false,  // mandatory
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(body),
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s", body)
}

func newWorker() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// q, err := ch.QueueDeclare(
	// 	"hello", // name
	// 	false,   // durable -- change to true to make sure task still exist after rabbitmq node restarts
	// 	false,   // delete when unused
	// 	false,   // exclusive
	// 	false,   // no-wait
	// 	nil,     // arguments
	// )
	// failOnError(err, "Failed to declare a queue")

	// because rabbitmq not allow change the existing queue, so workaround in other queue - durable true
	q, err := ch.QueueDeclare(
		"task_queue", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Failed to declare a queue")

	//

	// fair dispatch: this tells RabbitMQ not to give more than one message to a
	// worker at a time. Or, in other words, don't dispatch a new message to a
	// worker until it has processed and acknowledged the previous one. Instead,
	// it will dispatch it to the next worker that is not still busy.
	//
	// prefetch count = 1
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack -- change to false to ack manually
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			dotCount := bytes.Count(d.Body, []byte("."))
			t := time.Duration(dotCount)
			time.Sleep(t * time.Second)

			// Using this code, you can ensure that even if you terminate a worker
			// using CTRL+C while it was processing a message, nothing is lost. Soon
			// after the worker terminates, all unacknowledged messages are
			// redelivered. If auto ack is false, need ack manually as below
			//
			// sudo rabbitmqctl list_queues name messages_ready messages_unacknowledged
			d.Ack(false)

			log.Printf("Done")
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// run work queue example
func RunWorkerRMQ(skip bool) {
	if skip {
		return
	}

	if len(os.Args) < 2 {
		// publisher
		newTask()
		return
	}

	flag.NewFlagSet("c", flag.ContinueOnError)
	if os.Args[1] != "c" {
		log.Println("Nothing run")
		return
	}

	// consumer
	newWorker()
}

// MARK: PUB/SUB

// Example of pubsub fanout the logs
func RunPubSubFanoutRMQ(skip bool) {
	if skip {
		return
	}

	if len(os.Args) < 2 {
		// publisher
		emitLog()
		return
	}

	flag.NewFlagSet("c", flag.ContinueOnError)
	if os.Args[1] != "c" {
		log.Println("Nothing run")
		return
	}

	// consumer
	receiveLog()
}

func emitLog() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// declare the exchange fanout
	err = ch.ExchangeDeclare(
		"logs",   // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body := "Hello World!"
	err = ch.PublishWithContext(ctx,
		"logs", // exchange
		"",     // routing key - ignored for fanout
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	failOnError(err, "Failed to publish a message")

	log.Printf(" [x] Sent %s", body)
}

func receiveLog() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"logs",   // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare(
		"",    // name - ignore and let rabbitmq generate a random name
		false, // durable - no persistence
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// bind queue to exchange
	err = ch.QueueBind(
		q.Name, // queue name
		"",     // routing key
		"logs", // exchange
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")

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

	forever := make(chan struct{})
	go func() {
		for d := range msgs {
			log.Printf(" [x] %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	<-forever
}

// MARK: ROUTING

// MARK: TOPICS

// MARK: RPC
