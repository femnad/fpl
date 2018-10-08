package main

import (
	"path/filepath"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/femnad/mare"
	"github.com/google/uuid"
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

func ensureFileDir(fileName string) {
	fileDir := filepath.Dir(fileName)
	os.MkdirAll(fileDir, 0755)
}

func consumeBufferToFile(buffer *bytes.Buffer, fileName string) {
	ensureFileDir(fileName)

	f, err := os.OpenFile(fileName, os.O_CREATE | os.O_RDWR, 0644)
	mare.PanicIfErr(err)
	defer f.Close()

	for {
		line, err := buffer.ReadString('\n')
		if err != nil {
			mare.PanicIfNotOfType(err, io.EOF)
			break
		}
		_, err = f.WriteString(line)
		mare.PanicIfErr(err)
	}
}

func getUUID() string {
	uuid, err := uuid.NewUUID()
	mare.PanicIfErr(err)
	return uuid.String()
}

func handleOutput(stdOut, stdErr *bytes.Buffer, cfg *config) {
	fileName := getUUID()
	stdOutFile := fmt.Sprintf("%s/%s.out", cfg.outputDir, fileName)
	consumeBufferToFile(stdOut, stdOutFile)

	stdErrFile := fmt.Sprintf("%s/%s.err", cfg.outputDir, fileName)
	consumeBufferToFile(stdErr, stdErrFile)
}

func executeCommand(commandList []string, cfg *config) {
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

	handleOutput(&stdOut, &stdErr, cfg)
}

func consumeMessages(messages <-chan amqp.Delivery, cfg *config) {
	for d := range messages {
		commandList := getCommand(d)
		executeCommand(commandList, cfg)
	}
}

func consume(host string, port int, queueName string, cfg *config) {
	address := fmt.Sprintf("amqp://%s:%d", host, port)
	conn, err := amqp.Dial(address)
	mare.PanicIfErr(err)
	defer conn.Close()

	ch, err := conn.Channel()
	mare.PanicIfErr(err)
	defer ch.Close()

	q, err := ch.QueueDeclare(queueName, false, false, false, false, nil)
	mare.PanicIfErr(err)

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	mare.PanicIfErr(err)

	forever := make(chan bool)

	go consumeMessages(msgs, cfg)

	log.Printf("Waiting to consume messages")
	<-forever
}

type config struct {
	outputDir string
}

func getDefaultConfig() *config {
	cfg := config{outputDir: "/var/log/fpl/tasks"}
	return &cfg
}

func main() {
	host := flag.String("host", "localhost", "Host for RabbitMQ server")
	port := flag.Int("port", 5672, "Port for RabbitMQ server")
	queueName := flag.String("queue", "default", "Queue name")
	flag.Parse()

	cfg := getDefaultConfig()
	consume(*host, *port, *queueName, cfg)
}
