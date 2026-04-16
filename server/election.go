package server

import (
	"context"

	"github.com/oyavri/pi-bully/election"
	pb "github.com/oyavri/pi-bully/gen/bully"
	"go.uber.org/zap"
)

type ElectionServer struct {
	pb.UnimplementedElectionServiceServer
	engine election.Engine
	logger *zap.Logger
}

func NewElectionServer(engine election.Engine, logger *zap.Logger) *ElectionServer {
	return &ElectionServer{engine: engine, logger: logger}
}

func (s *ElectionServer) StartElection(ctx context.Context, req *pb.ElectionRequest) (*pb.ElectionResponse, error) {
	s.logger.Info("received election request",
		zap.Uint64("fromID", req.FromId),
		zap.Uint64("term", req.Term),
	)
	term := s.engine.OnStartElection(req.FromId, req.Term)
	return &pb.ElectionResponse{Ok: true, Term: term}, nil
}

func (s *ElectionServer) AnnounceLeader(ctx context.Context, req *pb.CoordinatorRequest) (*pb.CoordinatorResponse, error) {
	s.logger.Info("received announce leader",
		zap.Uint64("leaderID", req.LeaderId),
		zap.Uint64("term", req.Term),
	)
	s.engine.OnAnnounceLeader(req.LeaderId, req.Term)
	return &pb.CoordinatorResponse{}, nil
}
