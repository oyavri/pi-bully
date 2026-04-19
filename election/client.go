package election

import (
	"context"
	"fmt"

	pb "github.com/oyavri/pi-bully/gen/bully"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client interface {
	StartElection(ctx context.Context, addr string, req *pb.ElectionRequest) (*pb.ElectionResponse, error)
	AnnounceLeader(ctx context.Context, addr string, req *pb.CoordinatorRequest) error
}

type grpcClient struct {
	selfAddr string
}

func NewClient(selfAddr string) Client {
	return &grpcClient{selfAddr: selfAddr}
}

func (c *grpcClient) StartElection(ctx context.Context, addr string, req *pb.ElectionRequest) (*pb.ElectionResponse, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("start election: dial %s: %w", addr, err)
	}
	defer conn.Close()

	resp, err := pb.NewElectionServiceClient(conn).StartElection(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("start election: call %s: %w", addr, err)
	}

	return resp, nil
}

func (c *grpcClient) AnnounceLeader(ctx context.Context, addr string, req *pb.CoordinatorRequest) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("announce leader: dial %s: %w", addr, err)
	}
	defer conn.Close()

	_, err = pb.NewElectionServiceClient(conn).AnnounceLeader(ctx, req)
	if err != nil {
		return fmt.Errorf("announce leader: call %s: %w", addr, err)
	}

	return nil
}
