package services

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type AgentFSListRequest struct {
	ID        string
	AgentID   uuid.UUID
	Path      string
	Limit     int
	CreatedAt time.Time
	resultCh  chan AgentFSListResult
}

type AgentFSListResult struct {
	Response *FSListResponse
	Error    string
}

type AgentFSRequestBroker struct {
	mu      sync.Mutex
	byAgent map[uuid.UUID][]*AgentFSListRequest
	byID    map[string]*AgentFSListRequest
	ttl     time.Duration
}

func NewAgentFSRequestBroker() *AgentFSRequestBroker {
	return &AgentFSRequestBroker{
		byAgent: make(map[uuid.UUID][]*AgentFSListRequest),
		byID:    make(map[string]*AgentFSListRequest),
		ttl:     2 * time.Minute,
	}
}

func (b *AgentFSRequestBroker) Enqueue(agentID uuid.UUID, path string, limit int) (*AgentFSListRequest, <-chan AgentFSListResult) {
	if limit <= 0 || limit > 2000 {
		limit = 200
	}

	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cleanupLocked(now)

	req := &AgentFSListRequest{
		ID:        uuid.NewString(),
		AgentID:   agentID,
		Path:      path,
		Limit:     limit,
		CreatedAt: now,
		resultCh:  make(chan AgentFSListResult, 1),
	}

	b.byAgent[agentID] = append(b.byAgent[agentID], req)
	b.byID[req.ID] = req

	return req, req.resultCh
}

func (b *AgentFSRequestBroker) PopNext(agentID uuid.UUID) *AgentFSListRequest {
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cleanupLocked(now)

	queue := b.byAgent[agentID]
	if len(queue) == 0 {
		return nil
	}

	req := queue[0]
	if len(queue) == 1 {
		delete(b.byAgent, agentID)
	} else {
		b.byAgent[agentID] = queue[1:]
	}

	return req
}

func (b *AgentFSRequestBroker) Resolve(agentID uuid.UUID, requestID string, resp *FSListResponse, errorText string) bool {
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cleanupLocked(now)

	req, ok := b.byID[requestID]
	if !ok {
		return false
	}
	if req.AgentID != agentID {
		return false
	}

	delete(b.byID, requestID)
	req.resultCh <- AgentFSListResult{Response: resp, Error: errorText}
	close(req.resultCh)
	return true
}

func (b *AgentFSRequestBroker) Cancel(agentID uuid.UUID, requestID string) {
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cleanupLocked(now)

	req, ok := b.byID[requestID]
	if !ok || req.AgentID != agentID {
		return
	}

	delete(b.byID, requestID)
	close(req.resultCh)
}

func (b *AgentFSRequestBroker) cleanupLocked(now time.Time) {
	if b.ttl <= 0 {
		return
	}

	for id, req := range b.byID {
		if now.Sub(req.CreatedAt) <= b.ttl {
			continue
		}
		delete(b.byID, id)
		close(req.resultCh)
	}

	for agentID, queue := range b.byAgent {
		filtered := queue[:0]
		for _, req := range queue {
			if _, ok := b.byID[req.ID]; ok {
				filtered = append(filtered, req)
			}
		}
		if len(filtered) == 0 {
			delete(b.byAgent, agentID)
			continue
		}
		b.byAgent[agentID] = filtered
	}
}
