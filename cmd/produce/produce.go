package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/femnad/mare"
	"github.com/streadway/amqp"
)

func getCommand(commandList []string) string {
	return strings.Join(commandList, " ")
}

func getConnection(host string, port int) amqp.Connection {
	address := fmt.Sprintf("amqp://%s:%d", host, port)
	conn, err := amqp.Dial(address)
	mare.PanicIfErr(err)
	return *conn
}

func getChannel(connection amqp.Connection) amqp.Channel {
	ch, err := connection.Channel()
	mare.PanicIfErr(err)
	return *ch
}

func getPublishing(body string) amqp.Publishing {
	return amqp.Publishing{ContentType: "text/plain", Body: []byte(body)}
}

func declareQueue(ch amqp.Channel, queueName string) amqp.Queue {
	q, err := ch.QueueDeclare(queueName, false, false, false, false, nil)
	mare.PanicIfErr(err)
	return q
}

func publishMessage(channel amqp.Channel, queue amqp.Queue, message string) {
	publishing := getPublishing(message)
	err := channel.Publish("", queue.Name, false, false, publishing)
	mare.PanicIfErr(err)
}

func produceMessage(host string, port int, queueName string, commandList []string) {
	command := getCommand(commandList)

	conn := getConnection(host, port)
	defer conn.Close()

	ch := getChannel(conn)
	defer ch.Close()

	q := declareQueue(ch, queueName)

	publishMessage(ch, q, command)
}

func main() {
	host := flag.String("host", "localhost", "Host for RabbitMQ server")
	port := flag.Int("port", 5672, "Port for RabbitMQ server")
	queueName := flag.String("queue", "default", "Name for the queue for producing messages")

	flag.Parse()
	commandString := flag.Args()

	if len(commandString) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	produceMessage(*host, *port, *queueName, commandString)
}
