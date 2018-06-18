package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/femnad/mare"
	"github.com/streadway/amqp"
)

func getCommand(delivery amqp.Delivery) []string {
	commandString := string(delivery.Body)
	log.Printf("Got command: %s\n", commandString)
	return strings.Split(commandString, " ")
}

func handleWait(cmd exec.Cmd) {
	err := cmd.Wait()
	if err == nil {
		log.Printf("Command exited normally\n")
	} else {
		fmt.Printf("Command exited with %s\n", err)
	}
}

func handleOutput(stdOut, stdErr bytes.Buffer) {
	stdoutOutput := stdOut.String()
	if stdoutOutput != "" {
		log.Printf("Stdout output is:\n%s", stdOut.String())
	}
	stderrOutput := stdErr.String()
	if stderrOutput != "" {
		log.Printf("Stderr output is:\n%s", stdErr.String())
	}
}

func executeCommand(commandList []string) {
	cmd := exec.Command(commandList[0], commandList[1:]...)

	var stdOut bytes.Buffer
	var stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	err := cmd.Start()
	if err != nil {
		log.Printf("Error when running command %s", err)
		return
	}

	handleWait(*cmd)

	handleOutput(stdOut, stdErr)
}

func consumeMessages(messages <-chan amqp.Delivery) {
	for d := range messages {
		commandList := getCommand(d)
		executeCommand(commandList)
	}
}

func main() {
	host := flag.String("host", "localhost", "Host for RabbitMQ server")
	port := flag.Int("port", 5672, "Port for RabbitMQ server")
	flag.Parse()

	address := fmt.Sprintf("amqp://%s:%d", *host, *port)
	conn, err := amqp.Dial(address)
	mare.PanicIfErr(err)
	defer conn.Close()

	ch, err := conn.Channel()
	mare.PanicIfErr(err)
	defer ch.Close()

	q, err := ch.QueueDeclare("cmds", false, false, false, false, nil)
	mare.PanicIfErr(err)

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	mare.PanicIfErr(err)

	forever := make(chan bool)

	go consumeMessages(msgs)

	log.Printf("Waiting to consume messages")
	<-forever
}
