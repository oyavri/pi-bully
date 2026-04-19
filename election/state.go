package election

import "sync"

type Phase uint8

const (
	PhaseIdle Phase = iota
	PhaseElecting
	PhaseWaitingCoordinator
	PhaseLeader // indicates that there is a leader
)

type State struct {
	mu       sync.RWMutex
	leaderID uint64
	term     uint64
	phase    Phase
}

func NewState() *State {
	return &State{
		phase: PhaseIdle,
	}
}

func (s *State) IsLeader(selfID uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.leaderID == selfID
}

func (s *State) CurrentLeader() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.leaderID
}

func (s *State) Term() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.term
}

func (s *State) Phase() Phase {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.phase
}

// Updates local term if the incoming term is higher
func (s *State) UpdateTerm(term uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if term > s.term {
		s.term = term
		s.leaderID = 0
		s.phase = PhaseIdle
	}
}

// Starts a new election unless one is already active
func (s *State) BeginElection() (uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch s.phase {
	case PhaseElecting, PhaseWaitingCoordinator:
		return 0, false
	}

	s.leaderID = 0 // clean up the old leader ID
	s.term++
	s.phase = PhaseElecting
	return s.term, true
}

// Moves from actively electing to waiting for coordinator for the same term
func (s *State) EnterWaitingCoordinator(term uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.term != term {
		return false
	}
	if s.phase != PhaseElecting {
		return false
	}

	s.phase = PhaseWaitingCoordinator
	return true
}

// Applies leader state if term is not stale
func (s *State) SetLeader(leaderID uint64, term uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if term < s.term {
		return false
	}

	if term == s.term && s.leaderID != 0 && leaderID < s.leaderID {
		return false
	}

	s.term = term
	s.leaderID = leaderID
	s.phase = PhaseLeader
	return true
}

// Cleans up the leader only if it matches the expected leader
func (s *State) ClearLeaderIfMatches(leaderID uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.leaderID != leaderID {
		return false
	}

	s.leaderID = 0
	if s.phase == PhaseLeader {
		s.phase = PhaseIdle
	}
	return true
}

// Resets an active election if the term still matches
func (s *State) ResetElection(term uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.term != term {
		return false
	}

	switch s.phase {
	case PhaseElecting, PhaseWaitingCoordinator:
		s.phase = PhaseIdle
		return true
	}

	return false
}

func (s *State) InElection() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.phase == PhaseElecting || s.phase == PhaseWaitingCoordinator
}
