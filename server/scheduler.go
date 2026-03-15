package server

import (
	"context"

	pb "github.com/oyavri/pi-bully/gen/bully"
	"go.uber.org/zap"
)

type SchedulerServer struct {
	pb.UnimplementedSchedulerServiceServer
	logger *zap.Logger
}

func NewSchedulerServer(logger *zap.Logger) *SchedulerServer {
	return &SchedulerServer{logger: logger}
}

func (s *SchedulerServer) ReportResult(ctx context.Context, req *pb.ReportResultRequest) (*pb.ReportResultResponse, error) {
	s.logger.Info("received task result",
		zap.String("taskID", req.TaskId),
		zap.Uint64("workerID", req.WorkerId),
		zap.String("outcome", req.Outcome.String()),
	)

	return &pb.ReportResultResponse{}, nil
}
