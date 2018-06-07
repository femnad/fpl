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

	go func() {
		for d := range msgs {
			commandString := string(d.Body)
			log.Printf("Will run command: %s\n", commandString)
			commandList := strings.Split(commandString, " ")
			cmd := exec.Command(commandList[0], commandList[1:]...)
			var out bytes.Buffer
			var stdErr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &stdErr
			err := cmd.Start()
			if err != nil {
				log.Fatal(err)
			}
			err = cmd.Wait()
			if err == nil {
				log.Printf("Command exited with %s\n", err)
			}
			stdoutOutput := out.String()
			if stdoutOutput != "" {
				log.Printf("Stdout output is:\n%s", out.String())
			}
			stderrOutput := stdErr.String()
			if stderrOutput != "" {
				log.Printf("Stderr output is:\n%s", stdErr.String())
			}
		}
	}()

	log.Printf("Waiting to consume messages")
	<-forever
}
