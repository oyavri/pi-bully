package server

import (
	"context"

	"go.uber.org/zap"

	pb "github.com/oyavri/pi-bully/gen/bully"
	"github.com/oyavri/pi-bully/worker"
)

type TaskHandler interface {
	HandleAssignment(a worker.Assignment)
}

type WorkerServer struct {
	pb.UnimplementedWorkerServiceServer
	handler TaskHandler
	logger  *zap.Logger
}

func NewWorkerServer(handler TaskHandler, logger *zap.Logger) *WorkerServer {
	return &WorkerServer{handler: handler, logger: logger}
}

func (s *WorkerServer) AssignTask(ctx context.Context, req *pb.AssignTaskRequest) (*pb.AssignTaskResponse, error) {
	s.logger.Info("received task assignment",
		zap.String("taskID", req.TaskId),
		zap.String("executableURI", req.ExecutableUri),
		zap.String("inputURI", req.InputUri),
		zap.String("outputURI", req.OutputUri),
		zap.Strings("args", req.Args),
	)

	assignment := worker.Assignment{
		TaskID:        req.TaskId,
		ExecutableURI: req.ExecutableUri,
		InputURI:      req.InputUri,
		OutputURI:     req.OutputUri,
		Args:          req.Args,
	}

	s.handler.HandleAssignment(assignment)
	return &pb.AssignTaskResponse{Accepted: true}, nil
}
