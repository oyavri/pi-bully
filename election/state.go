package election

import "sync"

type State struct {
	mu         sync.RWMutex
	leaderID   uint64
	term       uint64
	inElection bool
}

func NewState() *State {
	return &State{}
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

// Starts an election and if successful, returns (new term, true)
// If an there is an ongoing election, returns (0, false)
func (s *State) BeginElection() (uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.inElection {
		return 0, false
	}
	s.inElection = true
	s.term++
	return s.term, true
}

func (s *State) SetLeader(leaderID uint64, term uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leaderID = leaderID
	s.term = term
	s.inElection = false
}

// clears in-progress flag without setting a leader
// used when a retry is needed
func (s *State) AbortElection() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inElection = false
}

func (s *State) InElection() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inElection
}
