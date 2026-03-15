package server

import (
	"context"

	"go.uber.org/zap"

	pb "github.com/oyavri/pi-bully/gen/bully"
)

type WorkerServer struct {
	pb.UnimplementedWorkerServiceServer
	logger *zap.Logger
}

func NewWorkerServer(logger *zap.Logger) *WorkerServer {
	return &WorkerServer{logger: logger}
}

func (s *WorkerServer) AssignTask(ctx context.Context, req *pb.AssignTaskRequest) (*pb.AssignTaskResponse, error) {
	s.logger.Info("received task assignment",
		zap.String("taskID", req.TaskId),
		zap.String("executableURI", req.ExecutableUri),
		zap.String("inputURI", req.InputUri),
		zap.String("outputURI", req.OutputUri),
		zap.Strings("args", req.Args),
	)

	return &pb.AssignTaskResponse{Accepted: true}, nil
}
