package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/kubernetes-sigs/maintainers/experimental/ksandbox/pkg/client"
	protocol "github.com/kubernetes-sigs/maintainers/experimental/ksandbox/protocol/ksandbox/v1alpha1"
	"google.golang.org/protobuf/encoding/prototext"
)

func main() {
	ctx := context.Background()
	err := run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	namespace := "default"
	image := ""
	flag.StringVar(&image, "image", image, "image to execute")
	buildAgentImage := ""
	flag.StringVar(&buildAgentImage, "agent", buildAgentImage, "name of build agent image")

	flag.Parse()

	if image == "" {
		return fmt.Errorf("must specify --image")
	}
	if buildAgentImage == "" {
		return fmt.Errorf("must specify --agent")
	}
	command := flag.Args()
	if len(command) == 0 {
		return fmt.Errorf("must specify command after flags")
	}

	// We assume this is being run on a developer machine (it's a test program),
	// rather than in-cluster.
	usePortForward := true
	c, err := client.NewAgentClient(ctx, namespace, buildAgentImage, image, usePortForward)
	if err != nil {
		return fmt.Errorf("error building agent client: %w", err)
	}
	defer c.Close()

	request := &protocol.ExecuteCommandRequest{
		Command: command,
	}
	response, err := c.ExecuteCommand(ctx, request)
	if err != nil {
		return fmt.Errorf("error executing in buildagent: %w", err)
	}

	fmt.Printf("response: %s", prototext.Format(response))

	return nil
}
