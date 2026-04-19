package strategy

import (
	"slices"
	"sync"

	"github.com/oyavri/pi-bully/cluster"
)

type RoundRobin struct {
	mu      sync.Mutex
	counter uint64
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func (r *RoundRobin) Pick(members map[uint64]cluster.Member) (cluster.Member, bool) {
	if len(members) == 0 {
		return cluster.Member{}, false
	}

	ids := make([]uint64, 0, len(members))
	for id := range members {
		ids = append(ids, id)
	}

	slices.Sort(ids)

	r.mu.Lock()
	idx := r.counter % uint64(len(ids))
	r.counter++
	r.mu.Unlock()

	return members[ids[idx]], true
}
