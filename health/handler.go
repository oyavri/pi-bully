package health

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/oyavri/pi-bully/cluster"
	"github.com/oyavri/pi-bully/election"
	"go.uber.org/zap"
)

type NodeStatus string

const (
	StatusNoLeader           NodeStatus = "no_leader"
	StatusElectionInProgress NodeStatus = "election_in_progress"
	StatusWaitingCoordinator NodeStatus = "waiting_for_coordinator"
	StatusLeaderElected      NodeStatus = "leader_elected"
)

type response struct {
	NodeID   uint64            `json:"node_id"`
	IsLeader bool              `json:"is_leader"`
	LeaderID uint64            `json:"leader_id"`
	Term     uint64            `json:"term"`
	Status   NodeStatus        `json:"status"`
	Members  map[uint64]string `json:"members"`
}

type Handler struct {
	selfID  uint64
	engine  election.Engine
	cluster cluster.Cluster
	logger  *zap.Logger
}

func NewHandler(selfID uint64, engine election.Engine, cl cluster.Cluster, logger *zap.Logger) *Handler {
	return &Handler{
		selfID:  selfID,
		engine:  engine,
		cluster: cl,
		logger:  logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	leaderID := h.engine.CurrentLeader()
	phase := h.engine.Phase()

	var status NodeStatus
	switch phase {
	case election.PhaseElecting:
		status = StatusElectionInProgress
	case election.PhaseWaitingCoordinator:
		status = StatusWaitingCoordinator
	case election.PhaseLeader:
		status = StatusLeaderElected
	default:
		status = StatusNoLeader
	}

	members := h.cluster.Members()
	memberAddrs := make(map[uint64]string, len(members))
	for id, m := range members {
		memberAddrs[id] = m.GRPCAddr
	}

	resp := response{
		NodeID:   h.selfID,
		IsLeader: h.engine.IsLeader(),
		LeaderID: leaderID,
		Term:     h.engine.Term(),
		Status:   status,
		Members:  memberAddrs,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode health response", zap.Error(err))
	}
}

func Start(port string, handler *Handler, logger *zap.Logger) {
	mux := http.NewServeMux()
	mux.Handle("/health", handler)

	addr := fmt.Sprintf(":%s", port)
	logger.Info("health server listening", zap.String("addr", addr))

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
			logger.Error("health server error", zap.Error(err))
		}
	}()
}
