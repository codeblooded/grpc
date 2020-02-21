package main

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc/grpc/testctrl/kubernetes"
)

func main() {
	adapter := &kubernetes.Adapter{}
	if err := adapter.ConnectWithConfig("/usr/local/google/home/benreed/.kube/config"); err != nil {
		panic(err)
	}

	deploymentBuilder := kubernetes.NewDeploymentBuilder("nginx", kubernetes.ServerRole, "nginx:latest")
	deployment := deploymentBuilder.Deployment()
	fmt.Printf("Deployment: %v\n", deploymentBuilder.ContainerPorts())

	deployment, err := adapter.CreateDeployment(context.Background(), deployment)
	if err != nil {
		panic(err)
	}

	fmt.Printf("> Deployment created, deleting in 30 seconds.\r")
	time.Sleep(30 * time.Second)

	if err = adapter.DeleteDeployment(context.Background(), deployment); err != nil {
		panic(err)
	}

	fmt.Printf("> Deployment successfully created then deleted.\n")
}

