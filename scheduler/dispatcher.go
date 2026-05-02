package scheduler

import (
	"context"

	pb "github.com/oyavri/pi-bully/gen/bully"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCTaskDispatcher struct{}

func NewGRPCTaskDispatcher() *GRPCTaskDispatcher {
	return &GRPCTaskDispatcher{}
}

func (d *GRPCTaskDispatcher) AssignTask(ctx context.Context, addr string, req *pb.AssignTaskRequest) (*pb.AssignTaskResponse, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	client := pb.NewWorkerServiceClient(conn)
	return client.AssignTask(ctx, req)
}
