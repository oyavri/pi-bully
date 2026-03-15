package server

import (
	"context"

	pb "github.com/oyavri/pi-bully/gen/bully"
	"go.uber.org/zap"
)

type ElectionServer struct {
	pb.UnimplementedElectionServiceServer
	logger *zap.Logger
}

func NewElectionServer(logger *zap.Logger) *ElectionServer {
	return &ElectionServer{logger: logger}
}

func (s *ElectionServer) StartElection(ctx context.Context, req *pb.ElectionRequest) (*pb.ElectionResponse, error) {
	s.logger.Info("received election request",
		zap.Uint64("fromID", req.FromId),
		zap.Uint64("term", req.Term),
	)

	return &pb.ElectionResponse{}, nil
}

func (s *ElectionServer) AnnounceLeader(ctx context.Context, req *pb.CoordinatorRequest) (*pb.CoordinatorResponse, error) {
	s.logger.Info("received announce leader",
		zap.Uint64("leaderID", req.LeaderId),
		zap.Uint64("term", req.Term),
	)

	return &pb.CoordinatorResponse{}, nil
}
