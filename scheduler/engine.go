package scheduler

import (
	"context"
	"maps"
	"time"

	"github.com/oyavri/pi-bully/cluster"
	"github.com/oyavri/pi-bully/config"
	"github.com/oyavri/pi-bully/election"
	pb "github.com/oyavri/pi-bully/gen/bully"
	"github.com/oyavri/pi-bully/scheduler/strategy"
	"github.com/oyavri/pi-bully/task"
	"go.uber.org/zap"
)

type TaskDispatcher interface {
	AssignTask(ctx context.Context, addr string, req *pb.AssignTaskRequest) (*pb.AssignTaskResponse, error)
}

type Engine struct {
	selfID         uint64
	election       election.Engine
	cluster        cluster.Cluster
	store          task.Store
	taskDispatcher TaskDispatcher
	strategy       strategy.Strategy
	cfg            config.SchedulerConfig
	logger         *zap.Logger

	didLeaderRecovery bool
}

func NewEngine(
	selfID uint64,
	election election.Engine,
	cluster cluster.Cluster,
	store task.Store,
	taskDispatcher TaskDispatcher,
	strategy strategy.Strategy,
	cfg config.SchedulerConfig,
	logger *zap.Logger,
) *Engine {
	return &Engine{
		selfID:         selfID,
		election:       election,
		cluster:        cluster,
		store:          store,
		taskDispatcher: taskDispatcher,
		strategy:       strategy,
		cfg:            cfg,
		logger:         logger.With(zap.String("component", "scheduler")),
	}
}

func (e *Engine) Start(ctx context.Context) {
	go e.run(ctx)
}

func (e *Engine) run(ctx context.Context) {
	ticker := time.NewTicker(e.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.tick(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	if !e.election.IsLeader() {
		e.didLeaderRecovery = false
		return
	}

	alive := e.aliveWorkers()
	if len(alive) == 0 {
		return
	}

	if !e.didLeaderRecovery {
		e.recoverOnLeadership(ctx, alive)
		e.didLeaderRecovery = true
	}

	if _, err := e.store.RecoverExpiredLeases(ctx); err != nil {
		e.logger.Error("failed to recover expired leases", zap.Error(err))
		return
	}

	workers, err := e.freeWorkers(ctx, alive)
	if err != nil {
		e.logger.Error("failed to determine free workers", zap.Error(err))
		return
	}
	if len(workers) == 0 {
		return
	}

	claimCount := min(e.cfg.BatchSize, len(workers))
	if claimCount == 0 {
		return
	}

	tasks, err := e.store.ClaimQueued(ctx, e.selfID, claimCount)
	if err != nil {
		e.logger.Error("failed to claim queued tasks", zap.Error(err))
		return
	}
	if len(tasks) == 0 {
		return
	}

	e.dispatchClaimedTasks(ctx, tasks, workers)
}

func (e *Engine) recoverOnLeadership(ctx context.Context, alive map[uint64]cluster.Member) {
	if _, err := e.store.RecoverStaleScheduling(ctx); err != nil {
		e.logger.Error("failed to recover stale scheduling", zap.Error(err))
	}

	if _, err := e.store.RecoverDeadWorkerLeases(ctx, alive); err != nil {
		e.logger.Error("failed to recover dead worker leases", zap.Error(err))
	}
}

func (e *Engine) aliveWorkers() map[uint64]cluster.Member {
	members := e.cluster.Members()
	alive := make(map[uint64]cluster.Member, len(members))

	for id, m := range members {
		if id == e.selfID {
			continue
		}
		alive[id] = m
	}

	return alive
}

func (e *Engine) freeWorkers(ctx context.Context, alive map[uint64]cluster.Member) (map[uint64]cluster.Member, error) {
	if len(alive) == 0 {
		return alive, nil
	}

	leases, err := e.store.ActiveLeases(ctx)
	if err != nil {
		return nil, err
	}

	free := make(map[uint64]cluster.Member, len(alive))
	maps.Copy(free, alive)

	for _, lease := range leases {
		delete(free, lease.WorkerID)
	}

	return free, nil
}

func (e *Engine) dispatchClaimedTasks(ctx context.Context, tasks []task.Task, workers map[uint64]cluster.Member) {
	for _, t := range tasks {
		worker, ok := e.strategy.Pick(workers)
		if !ok {
			return
		}

		if err := e.store.MarkAssigned(ctx, t.ID, worker.ID); err != nil {
			e.logger.Error("failed to mark task assigned",
				zap.String("taskID", t.ID.String()),
				zap.Uint64("workerID", worker.ID),
				zap.Error(err),
			)
			continue
		}

		_, err := e.taskDispatcher.AssignTask(ctx, worker.GRPCAddr, &pb.AssignTaskRequest{
			TaskId:        t.ID.String(),
			ExecutableUri: t.Executable,
			InputUri:      t.InputURI,
			OutputUri:     t.OutputURI,
			Args:          t.Args,
		})
		if err != nil {
			e.logger.Error("failed to dispatch task to worker",
				zap.String("taskID", t.ID.String()),
				zap.Uint64("workerID", worker.ID),
				zap.String("grpcAddr", worker.GRPCAddr),
				zap.Error(err),
			)

			if err := e.store.MarkWorkerLost(ctx, t.ID, worker.ID, "dispatch failed"); err != nil {
				e.logger.Error("failed to mark worker lost after dispatch failure",
					zap.String("taskID", t.ID.String()),
					zap.Uint64("workerID", worker.ID),
					zap.Error(err),
				)
			}
			continue
		}

		e.logger.Info("task dispatched",
			zap.String("taskID", t.ID.String()),
			zap.Uint64("workerID", worker.ID),
			zap.String("grpcAddr", worker.GRPCAddr),
		)
	}
}
