package election

import (
	"context"
	"time"

	"github.com/oyavri/pi-bully/cluster"
	"github.com/oyavri/pi-bully/config"
	pb "github.com/oyavri/pi-bully/gen/bully"
	"go.uber.org/zap"
)

// This is the narrow interface the engine needs from the cluster.
// Keeps election decoupled from the full cluster.Cluster interface
type MemberSource interface {
	Members() map[uint64]cluster.Member
	EventC() <-chan cluster.ClusterEvent
}

type internalEventType uint8

const (
	internalCoordinator internalEventType = iota
	internalRetry
)

type internalEvent struct {
	typ      internalEventType
	term     uint64
	leaderID uint64
	nodeID   uint64
}

type Engine interface {
	Start(ctx context.Context)
	OnStartElection(fromID uint64, term uint64, fromAddr string) uint64
	OnAnnounceLeader(leaderID uint64, term uint64)
	IsLeader() bool
	CurrentLeader() uint64
	Term() uint64
	InElection() bool
	Phase() Phase
}

type engine struct {
	selfID    uint64
	selfAddr  string
	cfg       config.ElectionConfig
	state     *State
	client    Client
	cluster   MemberSource
	electC    chan uint64
	internalC chan internalEvent
	logger    *zap.Logger
}

func NewEngine(cfg config.ElectionConfig, selfID uint64, selfAddr string, cl MemberSource, client Client, logger *zap.Logger) Engine {
	return &engine{
		selfID:    selfID,
		selfAddr:  selfAddr,
		cfg:       cfg,
		state:     NewState(),
		client:    client,
		cluster:   cl,
		electC:    make(chan uint64, 1),
		internalC: make(chan internalEvent, 8),
		logger:    logger.With(zap.String("component", "election")),
	}
}

func (e *engine) Start(ctx context.Context) {
	go e.run(ctx)
	go e.startElection(ctx)
}

func (e *engine) IsLeader() bool {
	return e.state.IsLeader(e.selfID)
}

func (e *engine) CurrentLeader() uint64 {
	return e.state.CurrentLeader()
}

func (e *engine) Term() uint64 {
	return e.state.Term()
}

func (e *engine) InElection() bool {
	return e.state.InElection()
}

func (e *engine) Phase() Phase {
	return e.state.Phase()
}

// OnStartElection is called by ElectionServer when a lower node sends StartElection
// Engine responds with ok=true immediately (in the server) and trigger its own
// election asynchronously
func (e *engine) OnStartElection(fromID uint64, term uint64, fromAddr string) uint64 {
	e.state.UpdateTerm(term)

	if e.state.IsLeader(e.selfID) {
		go func() {
			timeoutCtx, cancel := context.WithTimeout(context.Background(), e.cfg.ElectionTimeout)
			defer cancel()
			interval := e.cfg.AnnounceRetryInterval
			for {
				err := e.client.AnnounceLeader(timeoutCtx, fromAddr, &pb.CoordinatorRequest{
					LeaderId: e.selfID,
					Term:     e.state.Term(),
				})
				if err == nil {
					return
				}
				select {
				case <-timeoutCtx.Done():
					return
				case <-time.After(interval):
					interval *= 2
					if interval > e.cfg.ElectionTimeout {
						interval = e.cfg.ElectionTimeout
					}
				}
			}
		}()
		return e.state.Term()
	}

	select {
	case e.electC <- fromID:
	default:
	}
	return e.state.Term()
}

// OnAnnounceLeader is called by ElectionServer when a coordinator message arrives.
func (e *engine) OnAnnounceLeader(leaderID uint64, term uint64) {
	log := e.log("OnAnnounceLeader")

	ok := e.state.SetLeader(leaderID, term)
	if !ok {
		log.Warn("leader announcement rejected as stale",
			zap.Uint64("leaderID", leaderID),
			zap.Uint64("term", term),
		)
		return
	}

	e.signalCoordinator(term, leaderID)

	log.Info("leader announced",
		zap.Uint64("leaderID", leaderID),
		zap.Uint64("term", term),
	)
}

func (e *engine) signalCoordinator(term uint64, leaderID uint64) {
	e.internalC <- internalEvent{
		typ:      internalCoordinator,
		term:     term,
		leaderID: leaderID,
	}
}

func (e *engine) signalRetry(term uint64, nodeID uint64) {
	select {
	case e.internalC <- internalEvent{
		typ:    internalRetry,
		term:   term,
		nodeID: nodeID,
	}:
	default:
	}
}

func (e *engine) run(ctx context.Context) {
	eventC := e.cluster.EventC()
	for {
		select {
		case event := <-eventC:
			switch event.Type {
			case cluster.EventJoin:
				e.handleNodeJoined(ctx, event.MemberID)
			case cluster.EventLeave:
				e.handleNodeLost(ctx, event.MemberID)
			}
		case fromID := <-e.electC:
			e.handleElectRequest(ctx, fromID)
		case <-ctx.Done():
			return
		}
	}
}

func (e *engine) handleNodeJoined(ctx context.Context, joinedID uint64) {
	log := e.log("handleNodeJoined").With(zap.Uint64("joinedID", joinedID))

	if joinedID == e.selfID {
		log.Debug("self joined, ignoring")
		return
	}

	leaderID := e.state.CurrentLeader()

	if leaderID == 0 {
		log.Info("no leader yet, starting election")
		go e.startElection(ctx)
		return
	}

	if joinedID > leaderID {
		log.Info("higher node joined, starting election",
			zap.Uint64("currentLeaderID", leaderID),
		)
		go e.startElection(ctx)
	}
}

func (e *engine) handleNodeLost(ctx context.Context, lostID uint64) {
	log := e.log("handleNodeLost").With(zap.Uint64("lostID", lostID))

	leaderID := e.state.CurrentLeader()

	// if lost node was the leader, start election
	if leaderID == lostID {
		cleared := e.state.ClearLeaderIfMatches(lostID)
		log.Info("leader lost, starting election",
			zap.Bool("clearedLeader", cleared),
			zap.Uint64("term", e.state.Term()),
		)
		go e.startElection(ctx)
		return
	}

	if e.state.Phase() == PhaseWaitingCoordinator && lostID > e.selfID {
		term := e.state.Term()
		log.Info("higher node lost while waiting for coordinator, signaling retry",
			zap.Uint64("lostID", lostID),
			zap.Uint64("term", term),
		)
		e.signalRetry(term, lostID)
	}
}

func (e *engine) handleElectRequest(ctx context.Context, fromID uint64) {
	e.log("handleElectRequest").Info(
		"received election request from lower node, starting own election",
		zap.Uint64("fromID", fromID),
	)
	go e.startElection(ctx)
}

func (e *engine) startElection(ctx context.Context) {
	log := e.log("startElection")

	term, ok := e.state.BeginElection()
	if !ok {
		log.Debug("election already in progress, skipping")
		return
	}

	log.Info("election started", zap.Uint64("term", term))

	members := e.cluster.Members()
	var higher []cluster.Member
	for _, m := range members {
		if m.ID > e.selfID {
			higher = append(higher, m)
		}
	}

	if len(higher) == 0 {
		log.Info("no higher nodes found, declaring self as leader")
		e.declareLeader(ctx, term)
		return
	}

	log.Info("sending election to higher nodes", zap.Int("count", len(higher)))

	if e.sendToHigher(ctx, term, higher) {
		if !e.state.EnterWaitingCoordinator(term) {
			log.Warn("failed to enter waiting-for-coordinator phase",
				zap.Uint64("term", term),
				zap.Uint64("currentTerm", e.state.Term()),
				zap.Uint8("phase", uint8(e.state.Phase())),
			)
			return
		}

		log.Info("higher node is alive, waiting for coordinator",
			zap.Uint64("term", term),
		)
		e.waitForCoordinator(ctx, term)
		return
	}

	log.Info("no higher node responded, declaring self as leader")
	e.declareLeader(ctx, term)
}

// sendToHigher sends StartElection to all higher nodes concurrently
// Returns true as soon as any node responds ok=true within ElectionTimeout
func (e *engine) sendToHigher(ctx context.Context, term uint64, higher []cluster.Member) bool {
	log := e.log("sendToHigher")

	timeoutCtx, cancel := context.WithTimeout(ctx, e.cfg.ElectionTimeout)
	defer cancel()

	resultC := make(chan bool, len(higher))
	req := &pb.ElectionRequest{
		FromId:   e.selfID,
		Term:     term,
		GrpcAddr: e.selfAddr,
	}

	for _, m := range higher {
		go func(m cluster.Member) {
			interval := e.cfg.SendRetryInterval

			for {
				resp, err := e.client.StartElection(timeoutCtx, m.GRPCAddr, req)
				if err == nil {
					log.Info("higher node responded to election request",
						zap.Uint64("memberID", m.ID),
						zap.Bool("ok", resp.Ok),
						zap.Uint64("term", resp.Term),
					)

					resultC <- resp.Ok
					return
				}

				log.Warn("failed to reach higher node, retrying",
					zap.Uint64("memberID", m.ID),
					zap.Error(err),
				)

				select {
				case <-timeoutCtx.Done():
					log.Warn("higher node unreachable within timeout",
						zap.Uint64("memberID", m.ID),
					)
					resultC <- false
					return
				case <-time.After(interval):
					if interval < e.cfg.ElectionTimeout {
						interval *= 2
						if interval > e.cfg.ElectionTimeout {
							interval = e.cfg.ElectionTimeout
						}
					}
				}
			}
		}(m)
	}

	for range higher {
		select {
		case ok := <-resultC:
			if ok {
				cancel()
				return true
			}

		case <-timeoutCtx.Done():
			return false
		}
	}

	return false
}

func (e *engine) waitForCoordinator(ctx context.Context, term uint64) {
	log := e.log("waitForCoordinator").With(zap.Uint64("term", term))

	timer := time.NewTimer(e.cfg.ElectionTimeout)
	defer timer.Stop()

	for {
		select {
		case ev := <-e.internalC:
			if ev.term < term {
				log.Debug("ignoring stale internal event",
					zap.Uint64("eventTerm", ev.term),
					zap.Uint64("waitTerm", term),
					zap.Uint8("eventType", uint8(ev.typ)),
				)
				continue
			}

			switch ev.typ {
			case internalCoordinator:
				currentLeader := e.state.CurrentLeader()
				currentTerm := e.state.Term()

				if currentLeader == 0 || currentTerm < ev.term {
					log.Info("coordinator signal received but state is not valid, restarting election",
						zap.Uint64("leaderID", currentLeader),
						zap.Uint64("currentTerm", currentTerm),
						zap.Uint64("eventTerm", ev.term),
					)
					e.state.ResetElection(term)
					go e.startElection(ctx)
					return
				}

				log.Info("coordinator received, election complete",
					zap.Uint64("leaderID", currentLeader),
					zap.Uint64("coordinatorTerm", currentTerm),
				)
				return
			case internalRetry:
				log.Info("retry requested while waiting for coordinator",
					zap.Uint64("nodeID", ev.nodeID),
					zap.Uint64("eventTerm", ev.term),
				)

				e.state.ResetElection(term)
				go e.startElection(ctx)
				return
			}
		case <-timer.C:
			log.Warn("timeout waiting for coordinator, restarting election")
			e.state.ResetElection(term)
			go e.startElection(ctx)
			return
		case <-ctx.Done():
			log.Info("context canceled while waiting for coordinator")
			e.state.ResetElection(term)
			return
		}
	}
}

func (e *engine) declareLeader(ctx context.Context, term uint64) {
	log := e.log("declareLeader")

	log.Info("declaring self as leader",
		zap.Uint64("term", term),
		zap.Uint64("currentTerm", e.state.Term()),
	)
	if !e.state.SetLeader(e.selfID, term) {
		log.Warn("failed to declare self as leader due to stale term",
			zap.Uint64("term", term),
			zap.Uint64("currentTerm", e.state.Term()),
		)
		return
	}

	members := e.cluster.Members()
	for _, m := range members {
		go func(m cluster.Member) {
			timeoutCtx, cancel := context.WithTimeout(ctx, e.cfg.ElectionTimeout)
			defer cancel()

			interval := e.cfg.AnnounceRetryInterval
			for {
				err := e.client.AnnounceLeader(timeoutCtx, m.GRPCAddr, &pb.CoordinatorRequest{
					LeaderId: e.selfID,
					Term:     term,
				})
				if err == nil {
					return
				}
				log.Error("failed to announce leader to member, retrying",
					zap.Uint64("memberID", m.ID),
					zap.Error(err),
				)

				select {
				case <-timeoutCtx.Done():
					log.Error("failed to announce leader to peer within timeout",
						zap.Uint64("peerID", m.ID),
					)
					return
				case <-time.After(interval):
					interval *= 2
					if interval > e.cfg.ElectionTimeout {
						interval = e.cfg.ElectionTimeout
					}
				}
			}
		}(m)
	}
}

func (e *engine) log(method string) *zap.Logger {
	return e.logger.With(zap.String("method", method))
}
