package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rclone-backup-web/hub/logging"
)

var agentWSUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Agent connections are authenticated via API key, so allow all origins.
		return true
	},
}

// AgentWebSocket upgrades an authenticated agent request to a WebSocket connection.
// The connection is used for bidirectional agent<->hub messaging (actions, logs, status updates).
func (h *Handler) AgentWebSocket(c *gin.Context) {
	agentID := c.MustGet("agent_id").(uuid.UUID)

	reqCtx := c.Request.Context()
	ws, err := agentWSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[AgentWS] upgrade failed agent=%s: %v", agentID, err)
		return
	}

	conn := h.wsService.Register(agentID, ws)
	defer h.wsService.UnregisterConn(agentID, conn)

	logging.Debugf("[AgentWS] upgraded agent=%s remote=%s", agentID.String(), strings.TrimSpace(c.Request.RemoteAddr))

	// Best-effort announce connection.
	_ = h.wsService.SendJSON(agentID, WSMessage{Type: WSMessageTypeHubPing})

	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[AgentWS] read failed agent=%s: %v", agentID, err)
			}
			return
		}

		rawStr := strings.TrimSpace(string(raw))
		if rawStr == "" {
			continue
		}

		var msg WSMessage
		if err := json.Unmarshal([]byte(rawStr), &msg); err != nil {
			logging.Debugf("[AgentWS] invalid json agent=%s bytes=%d err=%v", agentID.String(), len(rawStr), err)
			_ = h.wsService.SendJSON(agentID, WSMessage{
				Type: WSMessageTypeHubError,
				Data: json.RawMessage(`{"message":"invalid json"}`),
			})
			continue
		}

		logging.Debugf("[AgentWS] received agent=%s type=%s data_bytes=%d", agentID.String(), strings.TrimSpace(msg.Type), len(msg.Data))
		if err := h.handleAgentWSMessage(reqCtx, agentID, msg); err != nil {
			log.Printf("[AgentWS] message failed agent=%s type=%s: %v", agentID.String(), strings.TrimSpace(msg.Type), err)
			apiErr := err.Error()
			if typed, ok := err.(*APIError); ok {
				apiErr = typed.Message
			}
			payload, _ := json.Marshal(map[string]string{"message": apiErr})
			_ = h.wsService.SendJSON(agentID, WSMessage{
				Type: WSMessageTypeHubError,
				Data: payload,
			})
		}
	}
}

func (h *Handler) handleAgentWSMessage(ctx context.Context, agentID uuid.UUID, msg WSMessage) error {
	switch msg.Type {
	case WSMessageTypeAgentPong:
		return nil
	case WSMessageTypeAgentHello:
		// Best-effort: accept and ignore for now.
		return nil
	case WSMessageTypeAgentHeartbeat:
		var req HeartbeatRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return &APIError{Status: 400, Message: "Invalid heartbeat payload"}
		}

		resp, err := h.processAgentHeartbeat(ctx, agentID, req)
		if err != nil {
			return err
		}

		if logging.IsDebug() {
			types := make([]string, 0, len(resp.Actions))
			for _, action := range resp.Actions {
				actionType := strings.TrimSpace(action.Type)
				if actionType == "" {
					actionType = strings.TrimSpace(action.Action)
				}
				if strings.TrimSpace(action.ExecutionID) != "" {
					actionType = actionType + "(" + strings.TrimSpace(action.ExecutionID) + ")"
				}
				if actionType != "" {
					types = append(types, actionType)
				}
			}
			logging.Debugf("[AgentWS] heartbeat agent=%s status=%s actions=%d types=%s", agentID.String(), strings.TrimSpace(req.Status), len(resp.Actions), strings.Join(types, ","))
		}

		data, err := json.Marshal(resp)
		if err != nil {
			return err
		}

		return h.wsService.SendJSON(agentID, WSMessage{
			Type: WSMessageTypeHubActions,
			Data: data,
		})
	case WSMessageTypeExecutionUpdate:
		var req WSExecutionUpdate
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return &APIError{Status: 400, Message: "Invalid execution.update payload"}
		}

		logging.Debugf("[AgentWS] execution.update agent=%s execution=%s status=%s ended_at=%s", agentID.String(), strings.TrimSpace(req.ExecutionID), strings.TrimSpace(req.Status), strings.TrimSpace(req.EndedAt))

		execID, err := uuid.Parse(req.ExecutionID)
		if err != nil {
			return &APIError{Status: 400, Message: "Invalid execution ID"}
		}

		updateReq := UpdateExecutionRequest{
			Status:       strings.TrimSpace(req.Status),
			LogOutput:    req.LogOutput,
			ErrorMessage: req.ErrorMessage,
			EndedAt:      req.EndedAt,
		}
		if strings.TrimSpace(updateReq.ErrorMessage) == "" && strings.TrimSpace(updateReq.LogOutput) != "" {
			updateReq.ErrorMessage = updateReq.LogOutput
			updateReq.LogOutput = ""
		}
		return h.processExecutionUpdate(ctx, agentID, execID, updateReq)
	case WSMessageTypeExecutionLogs:
		var req WSExecutionLogs
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return &APIError{Status: 400, Message: "Invalid execution.logs payload"}
		}

		logging.Debugf("[AgentWS] execution.logs agent=%s execution=%s entries=%d", agentID.String(), strings.TrimSpace(req.ExecutionID), len(req.Logs))

		execID, err := uuid.Parse(req.ExecutionID)
		if err != nil {
			return &APIError{Status: 400, Message: "Invalid execution ID"}
		}

		logReq := StreamLogsRequest{
			Logs: make([]struct {
				Timestamp string `json:"timestamp"`
				Message   string `json:"message"`
			}, 0, len(req.Logs)),
		}
		for _, entry := range req.Logs {
			logReq.Logs = append(logReq.Logs, struct {
				Timestamp string `json:"timestamp"`
				Message   string `json:"message"`
			}{
				Timestamp: entry.Timestamp,
				Message:   entry.Message,
			})
		}

		return h.processExecutionLogs(ctx, agentID, execID, logReq)
	case WSMessageTypeFSListResult:
		var req AgentFSListResultRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return &APIError{Status: 400, Message: "Invalid fs.list.result payload"}
		}
		return h.processAgentFSListResult(agentID, req)
	case WSMessageTypeConfigSyncRequest:
		tasks, err := h.buildLegacyAgentTasks(ctx, agentID)
		if err != nil {
			return err
		}

		data, err := json.Marshal(WSConfigSyncResponse{Tasks: tasks})
		if err != nil {
			return err
		}

		return h.wsService.SendJSON(agentID, WSMessage{
			Type: WSMessageTypeConfigSyncResponse,
			Data: data,
		})
	default:
		return &APIError{Status: 400, Message: "Unknown message type"}
	}
}
