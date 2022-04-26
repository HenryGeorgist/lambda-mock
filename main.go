package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/USACE/filestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/usace/wat-api/utils"
	"github.com/usace/wat-api/wat"
	"golang.org/x/net/context"
)

// StartContainer uses the Go SDK to run Docker containers..option 1
func StartContainer(imageWithTag string) (string, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return "", err
	}
	reader, err := cli.ImagePull(ctx, imageWithTag, types.ImagePullOptions{})
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	io.Copy(os.Stdout, reader)
	var chc *container.HostConfig
	var nnc *network.NetworkingConfig
	var vp *v1.Platform

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        imageWithTag,
		Cmd:          []string{"./main"},
		Tty:          true,
		AttachStdout: true,
	}, chc, nnc, vp, "")
	if err != nil {
		return "", err
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, err
}

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
func pullMessage(msg *sqs.Message, fs filestore.FileStore) error {
	modelPayload := wat.ModelPayload{}
	err := json.Unmarshal([]byte(string(*msg.Body)), &modelPayload)
	if err != nil {
		fmt.Println("unidentified message:", err)
		return err
	}
	fmt.Println("message received", *msg.MessageId)
	output, err := fs.PutObject(modelPayload.EventConfiguration.OutputDestination+"/payload.yml", []byte(string(*msg.Body)))
	if err != nil {
		fmt.Println("failure to push payload to filestore:", err)
		return err
	}
	fmt.Println(output)
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
	fs, err := loader.InitStore()
	messages := make(chan *sqs.Message, 2)
	go pollMessages(messages, queue)

	for {
		for message := range messages {
			err = pullMessage(message, fs)
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
