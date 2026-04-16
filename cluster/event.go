package cluster

import (
	"maps"
	"strconv"
	"sync"

	"github.com/hashicorp/memberlist"
	"go.uber.org/zap"
)

type Member struct {
	ID       uint64
	GRPCAddr string
}

type EventHandler struct {
	mu      sync.RWMutex
	members map[uint64]Member
	lostC   chan uint64
	logger  *zap.Logger
}

func NewEventHandler(logger *zap.Logger) *EventHandler {
	return &EventHandler{
		members: make(map[uint64]Member),
		lostC:   make(chan uint64, 1),
		logger:  logger,
	}
}

func (h *EventHandler) LostC() <-chan uint64 {
	return h.lostC
}

func (h *EventHandler) NotifyJoin(node *memberlist.Node) {
	log := h.log("NotifyJoin").With(zap.String("name", node.Name))

	id, err := strconv.ParseUint(node.Name, 10, 64)
	if err != nil {
		log.Error("failed to parse node ID from name", zap.Error(err))
		return
	}

	meta, err := decodeNodeMeta(node.Meta)
	if err != nil {
		log.Error("failed to decode node meta", zap.Error(err))
		return
	}

	member := Member{ID: id, GRPCAddr: meta.GRPCAddr}

	h.mu.Lock()
	h.members[id] = member
	h.mu.Unlock()

	log.Info("member joined",
		zap.Uint64("memberID", id),
		zap.String("grpcAddr", meta.GRPCAddr),
	)
}

func (h *EventHandler) NotifyLeave(node *memberlist.Node) {
	log := h.log("NotifyLeave").With(zap.String("name", node.Name))

	id, err := strconv.ParseUint(node.Name, 10, 64)
	if err != nil {
		log.Error("failed to parse node ID from name", zap.Error(err))
		return
	}

	h.mu.Lock()
	delete(h.members, id)
	select {
	case h.lostC <- id:
	default:
	}
	h.mu.Unlock()

	log.Info("member left", zap.Uint64("memberID", id))
}

func (h *EventHandler) NotifyUpdate(node *memberlist.Node) {
	log := h.log("NotifyUpdate").With(zap.String("name", node.Name))

	id, err := strconv.ParseUint(node.Name, 10, 64)
	if err != nil {
		log.Error("failed to parse node ID from name", zap.Error(err))
		return
	}

	meta, err := decodeNodeMeta(node.Meta)
	if err != nil {
		log.Error("failed to decode node meta", zap.Error(err))
		return
	}

	h.mu.Lock()
	h.members[id] = Member{ID: id, GRPCAddr: meta.GRPCAddr}
	h.mu.Unlock()

	log.Info("member updated",
		zap.Uint64("memberID", id),
		zap.String("grpcAddr", meta.GRPCAddr),
	)
}

func (h *EventHandler) Members() map[uint64]Member {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// return a snapshot of current members
	return maps.Clone(h.members)
}

func (h *EventHandler) log(method string) *zap.Logger {
	return h.logger.With(zap.String("method", method))
}
