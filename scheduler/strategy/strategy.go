package strategy

import "github.com/oyavri/pi-bully/cluster"

type Strategy interface {
	Pick(members map[uint64]cluster.Member) (cluster.Member, bool)
}
