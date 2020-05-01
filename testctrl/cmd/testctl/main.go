package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	grpcPb "github.com/codeblooded/grpc-proto/genproto/grpc/testing"
	lrpb "google.golang.org/genproto/googleapis/longrunning"
	"github.com/golang/protobuf/jsonpb"
	svcpb "github.com/grpc/grpc/testctrl/proto/scheduling/v1"
	"google.golang.org/grpc"
)

const (
	Success int = iota
	FlagError
	ConnectionError
	SchedulingError
)

type cliFlags struct {
	address  string
	driver   string
	server   string
	clients  clientList
	scenario scenario
}

func (c *cliFlags) validate() error {
	if c.driver == "" {
		return errors.New("-driver is required to orchestrate the test, but missing")
	}

	if c.scenario.String() == "" {
		return errors.New("-scenario is required to configure the test, but missing")
	}

	return nil
}

type clientList struct {
	clients []string
}

func (cl *clientList) String() string {
	return fmt.Sprintf("%v", cl.clients)
}

func (cl *clientList) Set(client string) error {
	cl.clients = append(cl.clients, client)
	return nil
}

type scenario struct {
	proto grpcPb.Scenario
}

func (s *scenario) String() string {
	return fmt.Sprintf("%v", s.proto)
}

func (s *scenario) Set(scenarioJSON string) error {
	if scenarioJSON == "" {
		return errors.New("a valid scenario is required, but missing")
	}

	err := jsonpb.UnmarshalString(scenarioJSON, &s.proto)
	if err != nil {
		return fmt.Errorf("could not parse scenario json: %v", err)
	}

	return nil
}

func exit(code int, messageFmt string, args ...interface{}) {
	fmt.Printf(messageFmt+"\n", args...)
	os.Exit(code)
}

func connect(ctx context.Context, address string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Printf("dialing server at %v\n", address)
	return grpc.DialContext(dialCtx, address, grpc.WithInsecure(),
		grpc.WithBlock(), grpc.WithDisableRetry())
}

func newScheduleRequest(flags cliFlags) *svcpb.StartTestSessionRequest {
	var workers []*svcpb.Component
	if flags.server != "" {
		workers = append(workers, &svcpb.Component{
			ContainerImage: flags.server,
			Kind:           svcpb.Component_SERVER,
		})
	}
	for _, client := range flags.clients.clients {
		workers = append(workers, &svcpb.Component{
			ContainerImage: client,
			Kind:           svcpb.Component_CLIENT,
		})
	}

	return &svcpb.StartTestSessionRequest{
		Scenario: &flags.scenario.proto,
		Driver: &svcpb.Component{
			ContainerImage: flags.driver,
			Kind:           svcpb.Component_DRIVER,
		},
		Workers: workers,
	}
}

func schedule(ctx context.Context, client svcpb.SchedulingServiceClient, request *svcpb.StartTestSessionRequest) (*lrpb.Operation, error) {
	scheduleCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Printf("invoking RPC to schedule session with test %q\n", request.Scenario.Name)
	return client.StartTestSession(scheduleCtx, request)
}

func main() {
	flags := cliFlags{}
	flag.StringVar(&flags.address, "address", "127.0.0.1:50051", "Container image of a driver for testing")
	flag.StringVar(&flags.driver, "driver", "", "Container image of a driver for testing")
	flag.StringVar(&flags.server, "server", "", "Container image of a server for testing")
	flag.Var(&flags.clients, "client", "Container image of a client for testing")
	flag.Var(&flags.scenario, "scenario", "Scenario which configures the test as JSON")
	flag.Parse()

	if err := flags.validate(); err != nil {
		exit(FlagError, err.Error())
	}

	conn, err := connect(context.Background(), flags.address)
	if err != nil {
		exit(ConnectionError, "could not connect to server: %v", err)
	}
	defer conn.Close()

	scheduleClient := svcpb.NewSchedulingServiceClient(conn)

	request := newScheduleRequest(flags)
	operation, err := schedule(context.Background(), scheduleClient, request)
	if err != nil {
		exit(SchedulingError, "scheduling session failed: %v", err)
	}

	fmt.Printf("operation: %v\n", operation)
}

