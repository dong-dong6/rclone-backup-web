package services

import "encoding/json"

// WSMessage is the envelope for Hub <-> Agent WebSocket frames.
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

const (
	WSMessageTypeAgentHello         = "agent.hello"
	WSMessageTypeAgentHeartbeat     = "agent.heartbeat"
	WSMessageTypeHubActions         = "hub.actions"
	WSMessageTypeExecutionUpdate    = "execution.update"
	WSMessageTypeExecutionLogs      = "execution.logs"
	WSMessageTypeFSListResult       = "fs.list.result"
	WSMessageTypeConfigSyncRequest  = "config.sync.request"
	WSMessageTypeConfigSyncResponse = "config.sync.response"
	WSMessageTypeHubPing            = "hub.ping"
	WSMessageTypeAgentPong          = "agent.pong"
	WSMessageTypeHubError           = "hub.error"
)

type WSAgentHello struct {
	AgentVersion string `json:"agent_version,omitempty"`
	Hostname     string `json:"hostname,omitempty"`
	Platform     string `json:"platform,omitempty"`
}

type WSExecutionUpdate struct {
	ExecutionID  string `json:"execution_id"`
	Status       string `json:"status"`
	LogOutput    string `json:"log_output,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	EndedAt      string `json:"ended_at,omitempty"`
}

type WSLogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

type WSExecutionLogs struct {
	ExecutionID string       `json:"execution_id"`
	Logs        []WSLogEntry `json:"logs"`
}

type WSConfigSyncRequest struct{}

type WSConfigSyncResponse struct {
	Tasks []Task `json:"tasks"`
}
