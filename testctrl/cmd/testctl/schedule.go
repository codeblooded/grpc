package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	grpcPb "github.com/codeblooded/grpc-proto/genproto/grpc/testing"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	svcpb "github.com/grpc/grpc/testctrl/proto/scheduling/v1"
	lrpb "google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc"
)

// ScheduleFlags is the set of flags necessary to schedule test sessions.
type ScheduleFlags struct {
	address  string
	driver   string
	server   string
	clients  clientList
	scenario scenario
}

func (s *ScheduleFlags) validate() error {
	if s.driver == "" {
		return errors.New("-driver is required to orchestrate the test, but missing")
	}

	if s.scenario.String() == "<nil>" {
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
	proto *grpcPb.Scenario
}

// String returns a string representation of the proto.
func (sc *scenario) String() string {
	return fmt.Sprintf("%v", sc.proto)
}

// Set parses the JSON string into a protobuf, allowing the flag to be set as it
// is used.
func (sc *scenario) Set(scenarioJSON string) error {
	if scenarioJSON == "" {
		return errors.New("a valid scenario is required, but missing")
	}

	sc.proto = &grpcPb.Scenario{}
	err := jsonpb.UnmarshalString(scenarioJSON, sc.proto)
	if err != nil {
		return fmt.Errorf("could not parse scenario json: %v", err)
	}

	return nil
}

func connect(ctx context.Context, address string) (*grpc.ClientConn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Printf("dialing server at %v\n", address)
	return grpc.DialContext(dialCtx, address, grpc.WithInsecure(),
		grpc.WithBlock(), grpc.WithDisableRetry())
}

func newScheduleRequest(flags ScheduleFlags) *svcpb.StartTestSessionRequest {
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
		Scenario: flags.scenario.proto,
		Driver: &svcpb.Component{
			ContainerImage: flags.driver,
			Kind:           svcpb.Component_DRIVER,
		},
		Workers: workers,
	}
}

func startTestSession(ctx context.Context, client svcpb.SchedulingServiceClient, request *svcpb.StartTestSessionRequest) (*lrpb.Operation, error) {
	scheduleCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	fmt.Printf("invoking RPC to schedule session with test %q\n", request.Scenario.Name)
	return client.StartTestSession(scheduleCtx, request)
}

func Schedule(args []string) {
	flags := ScheduleFlags{}
	scheduleFlags := flag.NewFlagSet("testctl schedule", flag.ExitOnError)
	scheduleFlags.StringVar(&flags.address, "address", "127.0.0.1:50051", "host and port of the scheduling server")
	scheduleFlags.StringVar(&flags.driver, "driver", "", "container image with a driver for testing")
	scheduleFlags.StringVar(&flags.server, "server", "", "container image with a server for testing")
	scheduleFlags.Var(&flags.clients, "client", "container image with a client for testing")
	scheduleFlags.Var(&flags.scenario, "scenario", "protobuf which configures the test (as a JSON string)")
	scheduleFlags.Parse(args)

	if err := flags.validate(); err != nil {
		exit(FlagError, err.Error())
	}

	conn, err := connect(context.Background(), flags.address)
	if err != nil {
		exit(ConnectionError, "could not connect to server: %v", err)
	}
	defer conn.Close()

	scheduleClient := svcpb.NewSchedulingServiceClient(conn)
	operationsClient := lrpb.NewOperationsClient(conn)

	request := newScheduleRequest(flags)
	operation, err := startTestSession(context.Background(), scheduleClient, request)
	if err != nil {
		exit(SchedulingError, "scheduling session failed: %v", err)
	}

	fmt.Printf("operation: %v\n", proto.MarshalTextString(operation))

	for {
		operation, err := operationsClient.GetOperation(
			context.Background(), &lrpb.GetOperationRequest{Name: operation.Name})
		if err != nil {
			exit(5, "mheh: %v", err)
		}

		var metadata svcpb.TestSessionMetadata
		if err := proto.Unmarshal(operation.Metadata.GetValue(), &metadata); err == nil {
			event := metadata.LatestEvent
			fmt.Printf("%s [%s] %s\n", event.Time, event.Kind, event.Description)
		}

		if operation.Done {
			var result svcpb.TestSessionResult
			if err := proto.Unmarshal(operation.GetResponse().GetValue(), &result); err == nil {
				fmt.Printf("%s\n", result.DriverLogs)
			}

			break
		}
		time.Sleep(5 * time.Second)
	}
}
