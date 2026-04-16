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
	LostC() <-chan uint64
}

type Engine interface {
	Start(ctx context.Context)
	OnStartElection(fromID uint64, term uint64) uint64
	OnAnnounceLeader(leaderID uint64, term uint64)
	IsLeader() bool
	CurrentLeader() uint64
	Term() uint64
	InElection() bool
}

type engine struct {
	selfID    uint64
	cfg       config.ElectionConfig
	state     *State
	client    Client
	cluster   MemberSource
	electC    chan uint64   // receives fromID when lower node sends StartElection to this node
	announceC chan struct{} // wakes up a standing-down startElection goroutine
	logger    *zap.Logger
}

func NewEngine(cfg config.ElectionConfig, selfID uint64, cl MemberSource, client Client, logger *zap.Logger) Engine {
	return &engine{
		selfID:    selfID,
		cfg:       cfg,
		state:     NewState(),
		client:    client,
		cluster:   cl,
		electC:    make(chan uint64, 1),
		announceC: make(chan struct{}, 1),
		logger:    logger.With(zap.String("component", "election")),
	}
}

func (e *engine) Start(ctx context.Context) {
	go e.run(ctx)
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

// OnStartElection is called by ElectionServer when a lower node sends StartElection
// Engine responds with ok=true immediately (in the server) and trigger its own
// election asynchronously
func (e *engine) OnStartElection(fromID uint64, term uint64) uint64 {
	select {
	case e.electC <- fromID:
	default:
	}
	return e.state.Term()
}

// OnAnnounceLeader is called by ElectionServer when a coordinator message arrives.
// Always updates state and wakes up any standing-down goroutine
func (e *engine) OnAnnounceLeader(leaderID uint64, term uint64) {
	e.state.SetLeader(leaderID, term)
	select {
	case e.announceC <- struct{}{}:
	default:
	}
	e.log("OnAnnounceLeader").Info(
		"leader announced",
		zap.Uint64("leaderID", leaderID),
		zap.Uint64("term", term),
	)
}

func (e *engine) run(ctx context.Context) {
	lostC := e.cluster.LostC()
	for {
		select {
		case lostID := <-lostC:
			e.handleNodeLost(ctx, lostID)
		case fromID := <-e.electC:
			e.handleElectRequest(ctx, fromID)
		case <-ctx.Done():
			return
		}
	}
}

func (e *engine) handleNodeLost(ctx context.Context, lostID uint64) {
	log := e.log("handleNodeLost").
		With(zap.Uint64("lostID", lostID))

	if e.state.CurrentLeader() != lostID {
		log.Debug("lost node was not the leader, ignoring")
		return
	}

	log.Info("leader lost, starting election")
	go e.startElection(ctx)
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
		log.Info("higher node is alive, standing down")
		e.standDown(ctx, term)
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

	for _, m := range higher {
		go func(m cluster.Member) {
			resp, err := e.client.StartElection(timeoutCtx, m.GRPCAddr, &pb.ElectionRequest{
				FromId: e.selfID,
				Term:   term,
			})

			if err != nil {
				log.Warn("failed to reach higher node",
					zap.Uint64("memberID", m.ID),
					zap.Error(err),
				)
				resultC <- false
				return
			}
			resultC <- resp.Ok
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

// standDown waits for an AnnounceLeader within ElectionTimeout
// If none arrives, aborts and lets the next NotifyLeave retrigger
func (e *engine) standDown(ctx context.Context, term uint64) {
	log := e.log("standDown")

	// clear any stale wakeup signal before waiting
	select {
	case <-e.announceC:
	default:
	}

	select {
	case <-e.announceC:
		if e.state.Term() < term {
			log.Warn("received stale leader announcement, aborting")
			e.state.AbortElection()
			return
		}
		log.Info("leader announced, election complete")
	case <-time.After(e.cfg.ElectionTimeout):
		log.Warn("timeout waiting for leader announcement, aborting election")
		e.state.AbortElection()
	case <-ctx.Done():
		e.state.AbortElection()
	}
}

func (e *engine) declareLeader(ctx context.Context, term uint64) {
	log := e.log("declareLeader")

	e.state.SetLeader(e.selfID, term)
	log.Info("declaring self as leader", zap.Uint64("term", term))

	members := e.cluster.Members()
	for _, m := range members {
		go func(m cluster.Member) {
			if err := e.client.AnnounceLeader(ctx, m.GRPCAddr, &pb.CoordinatorRequest{
				LeaderId: e.selfID,
				Term:     term,
			}); err != nil {
				log.Error("failed to announce leader to member",
					zap.Uint64("memberID", m.ID),
					zap.Error(err),
				)
			}
		}(m)
	}
}

func (e *engine) log(method string) *zap.Logger {
	return e.logger.With(zap.String("method", method))
}
