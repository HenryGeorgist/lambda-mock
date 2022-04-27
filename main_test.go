package main

import (
	"fmt"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func TestRunContainer(t *testing.T) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	images, err := cli.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		panic(err)
	}

	for _, image := range images {
		fmt.Println(image.ID)
	}
	env := make([]string, 0)
	resp, err := StartContainer("docker.io/williamlehman/fragilitycurveplugin:v0.0.1", "", env)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	fmt.Println(resp)
}
