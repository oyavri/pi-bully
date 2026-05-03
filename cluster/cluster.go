package cluster

import (
	"fmt"
	"net"
	"strconv"

	"github.com/hashicorp/memberlist"
	"github.com/oyavri/pi-bully/config"
	"go.uber.org/zap"
)

type Cluster interface {
	Join(seeds []string) error
	Leave() error
	Members() map[uint64]Member
	MemberAddr(id uint64) (string, bool)
	EventC() <-chan ClusterEvent
}

type cluster struct {
	ml      *memberlist.Memberlist
	handler *EventHandler
	logger  *zap.Logger
}

func New(cfg config.MemberlistConfig, nodeCfg config.NodeConfig, serverCfg config.ServerConfig, logger *zap.Logger) (Cluster, error) {
	log := logger.With(zap.String("component", "cluster"))

	grpcAddr := nodeCfg.Address
	meta, err := encodeNodeMeta(NodeMeta{GRPCAddr: grpcAddr})
	if err != nil {
		return nil, fmt.Errorf("cluster: encode node meta: %w", err)
	}

	handler := NewEventHandler(log)

	advertiseAddr := cfg.AdvertiseAddr
	if ip := net.ParseIP(advertiseAddr); ip == nil {
		// it's a hostname, resolve it
		addrs, err := net.LookupHost(advertiseAddr)
		if err != nil {
			return nil, fmt.Errorf("cluster: resolve advertise addr %q: %w", advertiseAddr, err)
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("cluster: no addresses found for %q", advertiseAddr)
		}
		advertiseAddr = addrs[0]
	}

	mlCfg := memberlist.DefaultLANConfig()
	mlCfg.Name = strconv.FormatUint(nodeCfg.ID, 10)
	mlCfg.BindAddr = cfg.BindAddr
	mlCfg.BindPort = cfg.BindPort
	mlCfg.AdvertiseAddr = advertiseAddr
	mlCfg.ProbeInterval = cfg.ProbeInterval
	mlCfg.ProbeTimeout = cfg.ProbeTimeout
	mlCfg.GossipInterval = cfg.GossipInterval
	mlCfg.PushPullInterval = cfg.PushPullInterval
	mlCfg.SuspicionMult = cfg.SuspicionMult
	mlCfg.Events = handler

	mlCfg.Delegate = &metaDelegate{meta: meta}

	// Replace memberlist's logger with zap
	mlCfg.Logger = nil
	mlCfg.LogOutput = newZapWriter(log)

	ml, err := memberlist.Create(mlCfg)
	if err != nil {
		return nil, fmt.Errorf("cluster: create memberlist: %w", err)
	}

	return &cluster{
		ml:      ml,
		handler: handler,
		logger:  log,
	}, nil
}

func (c *cluster) Join(seeds []string) error {
	n, err := c.ml.Join(seeds)
	if err != nil {
		c.logger.Error("failed to join cluster", zap.Error(err))
		return fmt.Errorf("cluster: join: %w", err)
	}

	c.logger.Info("joined cluster", zap.Int("contactedNodes", n))
	return nil
}

func (c *cluster) Leave() error {
	if err := c.ml.Leave(0); err != nil {
		c.logger.Error("failed to leave cluster", zap.Error(err))
		return fmt.Errorf("cluster: leave: %w", err)
	}
	if err := c.ml.Shutdown(); err != nil {
		c.logger.Error("failed to shutdown memberlist", zap.Error(err))
		return fmt.Errorf("cluster: shutdown: %w", err)
	}
	c.logger.Info("left cluster")
	return nil
}

func (c *cluster) Members() map[uint64]Member {
	return c.handler.Members()
}

func (c *cluster) MemberAddr(id uint64) (string, bool) {
	members := c.handler.Members()
	member, ok := members[id]
	if !ok {
		return "", false
	}
	return member.GRPCAddr, true
}

func (c *cluster) EventC() <-chan ClusterEvent {
	return c.handler.EventC()
}

type metaDelegate struct {
	meta []byte
}

func (d *metaDelegate) NodeMeta(limit int) []byte {
	if len(d.meta) > limit {
		return nil
	}
	return d.meta
}

func (d *metaDelegate) NotifyMsg([]byte)                           {}
func (d *metaDelegate) GetBroadcasts(overhead, limit int) [][]byte { return nil }
func (d *metaDelegate) LocalState(join bool) []byte                { return nil }
func (d *metaDelegate) MergeRemoteState(buf []byte, join bool)     {}

type zapWriter struct {
	logger *zap.Logger
}

func newZapWriter(logger *zap.Logger) *zapWriter {
	return &zapWriter{logger: logger}
}

func (w *zapWriter) Write(p []byte) (int, error) {
	w.logger.Debug(string(p))
	return len(p), nil
}
