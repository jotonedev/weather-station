package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func run(args []string) error {
	fs := flag.NewFlagSet("healthprobe", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:8090", "target address")
	service := fs.String("service", "", "service name to check")
	timeout := fs.Duration("timeout", 3*time.Second, "check timeout")

	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("connect to %q: %w", *addr, err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: *service})
	if err != nil {
		return fmt.Errorf("health check %q: %w", *service, err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("service %q status is %s", *service, resp.Status)
	}

	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "health check failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
