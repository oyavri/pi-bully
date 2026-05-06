package worker

import (
	"context"
	"time"

	"github.com/oyavri/pi-bully/cluster"
	"github.com/oyavri/pi-bully/election"
	pb "github.com/oyavri/pi-bully/gen/bully"
	"go.uber.org/zap"
)

type WorkerClient interface {
	RenewLease(ctx context.Context, addr string, req *pb.RenewLeaseRequest) (*pb.RenewLeaseResponse, error)
	MarkRunning(ctx context.Context, addr string, req *pb.MarkRunningRequest) (*pb.MarkRunningResponse, error)
	ReportResult(ctx context.Context, addr string, req *pb.ReportResultRequest) (*pb.ReportResultResponse, error)
}

type Assignment struct {
	TaskID        string
	ExecutableURI string
	InputURI      string
	OutputURI     string
	Args          []string
}

type Engine struct {
	workerID        uint64
	election        election.Engine
	cluster         cluster.Cluster
	workerClient    WorkerClient
	executor        Executor
	leaseRenewEvery time.Duration
	rpcTimeout      time.Duration
	logger          *zap.Logger
}

func NewEngine(
	workerID uint64,
	election election.Engine,
	cluster cluster.Cluster,
	workerClient WorkerClient,
	executor Executor,
	leaseRenewEvery time.Duration,
	rpcTimeout time.Duration,
	logger *zap.Logger,
) *Engine {
	return &Engine{
		workerID:        workerID,
		election:        election,
		cluster:         cluster,
		workerClient:    workerClient,
		executor:        executor,
		leaseRenewEvery: leaseRenewEvery,
		rpcTimeout:      rpcTimeout,
		logger:          logger.With(zap.String("component", "worker")),
	}
}

func (e *Engine) HandleAssignment(a Assignment) {
	go e.runTask(a)
}

func (e *Engine) runTask(a Assignment) {
	taskID := a.TaskID

	addr, ok := e.leaderAddr()
	if !ok {
		e.logger.Error("failed to resolve leader address before starting task",
			zap.String("taskID", taskID),
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.rpcTimeout)
	_, err := e.workerClient.MarkRunning(ctx, addr, &pb.MarkRunningRequest{
		TaskId:   taskID,
		WorkerId: e.workerID,
	})
	cancel()
	if err != nil {
		e.logger.Error("failed to mark task running",
			zap.String("taskID", taskID),
			zap.Error(err),
		)
		return
	}

	e.logger.Info("task started",
		zap.String("taskID", taskID),
	)

	taskCtx, cancelTask := context.WithCancel(context.Background())
	defer cancelTask()

	go e.renewLoop(taskCtx, taskID)

	if err := e.executor.Execute(taskCtx, a); err != nil {
		e.logger.Error("task execution failed",
			zap.String("taskID", taskID),
			zap.Error(err),
		)

		addr, ok := e.leaderAddr()
		if !ok {
			e.logger.Error("failed to resolve leader address for failure reporting",
				zap.String("taskID", taskID),
			)
			return
		}

		ctx, cancel = context.WithTimeout(context.Background(), e.rpcTimeout)
		defer cancel()

		_, reportErr := e.workerClient.ReportResult(ctx, addr, &pb.ReportResultRequest{
			TaskId:   taskID,
			WorkerId: e.workerID,
			Outcome:  pb.TaskOutcome_FAILED,
			Error:    err.Error(),
		})
		if reportErr != nil {
			e.logger.Error("failed to report task failure",
				zap.String("taskID", taskID),
				zap.Error(reportErr),
			)
		}

		e.logger.Info("task failed",
			zap.String("taskID", taskID),
		)
		return
	}

	addr, ok = e.leaderAddr()
	if !ok {
		e.logger.Error("failed to resolve leader address for result reporting",
			zap.String("taskID", taskID),
		)
		return
	}

	ctx, cancel = context.WithTimeout(context.Background(), e.rpcTimeout)
	defer cancel()

	_, err = e.workerClient.ReportResult(ctx, addr, &pb.ReportResultRequest{
		TaskId:   taskID,
		WorkerId: e.workerID,
		Outcome:  pb.TaskOutcome_SUCCESS,
	})
	if err != nil {
		e.logger.Error("failed to report result",
			zap.String("taskID", taskID),
			zap.Error(err),
		)
		return
	}

	e.logger.Info("task completed",
		zap.String("taskID", taskID),
	)
}

func (e *Engine) renewLoop(ctx context.Context, taskID string) {
	ticker := time.NewTicker(e.leaseRenewEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			addr, ok := e.leaderAddr()
			if !ok {
				e.logger.Warn("failed to resolve leader address for lease renewal",
					zap.String("taskID", taskID),
				)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), e.rpcTimeout)
			_, err := e.workerClient.RenewLease(ctx, addr, &pb.RenewLeaseRequest{
				TaskId:   taskID,
				WorkerId: e.workerID,
			})
			cancel()

			if err != nil {
				e.logger.Warn("failed to renew lease",
					zap.String("taskID", taskID),
					zap.Error(err),
				)
				continue
			}

			e.logger.Debug("lease renewed",
				zap.String("taskID", taskID),
			)
		case <-ctx.Done():
			return
		}
	}
}

func (e *Engine) leaderAddr() (string, bool) {
	leaderID := e.election.CurrentLeader()
	if leaderID == 0 || e.election.IsLeader() {
		return "", false
	}

	return e.cluster.MemberAddr(leaderID)
}
