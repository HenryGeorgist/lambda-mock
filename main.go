package main

import (
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/usace/wat-api/utils"
	"github.com/usace/wat-api/wat"

	"fmt"
)

func pollMessages(chn chan<- *sqs.Message, queue *sqs.SQS) {

	for {
		output, err := queue.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queue.Endpoint + "/queue/messages"),
			MaxNumberOfMessages: aws.Int64(2),
			WaitTimeSeconds:     aws.Int64(5),
		})

		if err != nil {
			fmt.Println("failed to fetch sqs message", err)
		}

		for _, message := range output.Messages {
			chn <- message
		}

	}

}

// pullMessage...
func pullMessage(msg *sqs.Message) error {
	event := wat.EventConfiguration{}
	err := json.Unmarshal([]byte(string(*msg.Body)), &event)
	if err != nil {
		fmt.Println("unidentified message:", err)
		return err
	}
	fmt.Println("message received", *msg.MessageId)
	return nil
}

func deleteMessage(msg *sqs.Message, queue *sqs.SQS) error {
	_, err := queue.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queue.Endpoint + "/queue/messages"),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		fmt.Println("message not deleted: ", err)
		return err
	}
	fmt.Println("message deleted", *msg.MessageId)
	return err
}

func main() {
	fmt.Println("lambda-mock")
	fmt.Println("initializing a mock lambda")
	loader, err := utils.InitLoader("")
	if err != nil {
		log.Fatal(err)
		return
	}
	queue, err := loader.InitQueue()
	if err != nil {
		log.Fatal(err)
		return
	}

	messages := make(chan *sqs.Message, 2)
	go pollMessages(messages, queue)

	for {
		for message := range messages {
			err = pullMessage(message)
			if err != nil {
				fmt.Println(err)
			}
			err = deleteMessage(message, queue)
			if err != nil {
				fmt.Println(err)
			}
		}

	}

}
