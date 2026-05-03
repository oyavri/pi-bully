package server

import (
	"context"

	pb "github.com/oyavri/pi-bully/gen/bully"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WorkerClient struct{}

func NewWorkerClient() *WorkerClient {
	return &WorkerClient{}
}

func (c *WorkerClient) RenewLease(ctx context.Context, addr string, req *pb.RenewLeaseRequest) (*pb.RenewLeaseResponse, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewSchedulerServiceClient(conn)
	return client.RenewLease(ctx, req)
}

func (c *WorkerClient) ReportResult(ctx context.Context, addr string, req *pb.ReportResultRequest) (*pb.ReportResultResponse, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewSchedulerServiceClient(conn)
	return client.ReportResult(ctx, req)
}

func (c *WorkerClient) MarkRunning(ctx context.Context, addr string, req *pb.MarkRunningRequest) (*pb.MarkRunningResponse, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewSchedulerServiceClient(conn)
	return client.MarkRunning(ctx, req)
}
