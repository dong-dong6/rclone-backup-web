package api

import "encoding/json"

// WSMessage is the envelope for Hub <-> Agent WebSocket frames.
// The "type" field routes messages; "data" contains the typed payload.
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
	ExecutionID  string `json:"execution_id" binding:"required"`
	Status       string `json:"status" binding:"required"`
	LogOutput    string `json:"log_output,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	EndedAt      string `json:"ended_at,omitempty"`
}

type WSLogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

type WSExecutionLogs struct {
	ExecutionID string       `json:"execution_id" binding:"required"`
	Logs        []WSLogEntry `json:"logs"`
}

type WSConfigSyncRequest struct{}

// AgentLegacyTask matches the agent's legacy task list format (used by some agents and WS config sync).
// NOTE: "remote_config" contains the decrypted rclone config content (not base64).
type AgentLegacyTask struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Schedule     string   `json:"schedule"`
	RemoteConfig string   `json:"remote_config"`
	SourcePath   string   `json:"source_path"`
	DestPath     string   `json:"dest_path"`
	RcloneArgs   []string `json:"rclone_args"`
	Enabled      bool     `json:"enabled"`

	BackupMode          string `json:"backup_mode,omitempty"`
	ArchiveFormat       string `json:"archive_format,omitempty"`
	EncryptionEnabled   bool   `json:"encryption_enabled"`
	EncryptionPassword  string `json:"encryption_password,omitempty"`
	EncryptionPassword2 string `json:"encryption_password2,omitempty"`
}

type WSConfigSyncResponse struct {
	Tasks []AgentLegacyTask `json:"tasks"`
}
