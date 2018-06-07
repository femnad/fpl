package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/femnad/mare"
	"github.com/streadway/amqp"
)

func main() {
	host := flag.String("host", "localhost", "Host for RabbitMQ server")
	port := flag.Int("port", 5672, "Port for RabbitMQ server")
	flag.Parse()
	commandString := flag.Args()

	if len(commandString) == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	command := strings.Join(commandString, " ")
	address := fmt.Sprintf("amqp://%s:%d", *host, *port)
	conn, err := amqp.Dial(address)
	mare.PanicIfErr(err)
	defer conn.Close()

	ch, err := conn.Channel()
	mare.PanicIfErr(err)
	defer ch.Close()

	q, err := ch.QueueDeclare("cmds", false, false, false, false, nil)
	mare.PanicIfErr(err)
	err = ch.Publish("", q.Name, false, false,
		amqp.Publishing{ContentType: "text/plain",
			Body: []byte(command)})
	mare.PanicIfErr(err)
}
