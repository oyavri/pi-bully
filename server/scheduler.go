package server

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/oyavri/pi-bully/gen/bully"
	"github.com/oyavri/pi-bully/task"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SchedulerServer struct {
	pb.UnimplementedSchedulerServiceServer
	store  task.Store
	logger *zap.Logger
}

func NewSchedulerServer(store task.Store, logger *zap.Logger) *SchedulerServer {
	return &SchedulerServer{store: store, logger: logger}
}

func (s *SchedulerServer) ReportResult(ctx context.Context, req *pb.ReportResultRequest) (*pb.ReportResultResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	switch req.Outcome {
	case pb.TaskOutcome_SUCCESS:
		err = s.store.MarkCompleted(ctx, taskID, req.WorkerId)
	case pb.TaskOutcome_FAILED:
		err = s.store.MarkFailed(ctx, taskID, req.WorkerId)
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid outcome")
	}

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.ReportResultResponse{}, nil
}

func (s *SchedulerServer) RenewLease(ctx context.Context, req *pb.RenewLeaseRequest) (*pb.RenewLeaseResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	if err := s.store.RenewLease(ctx, taskID, req.WorkerId); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.RenewLeaseResponse{}, nil
}

func (s *SchedulerServer) MarkRunning(ctx context.Context, req *pb.MarkRunningRequest) (*pb.MarkRunningResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	if err := s.store.MarkRunning(ctx, taskID, req.WorkerId); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.MarkRunningResponse{}, nil
}
