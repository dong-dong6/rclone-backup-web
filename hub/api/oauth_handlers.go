package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createOAuthFlowRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scope        string `json:"scope"`
	Region       string `json:"region"`
}

type createOAuthFlowResponse struct {
	FlowID    string `json:"flow_id"`
	StartURL  string `json:"start_url"`
	ExpiresAt string `json:"expires_at"`
}

type oauthFlowStatusResponse struct {
	Status string                 `json:"status"`
	Token  map[string]interface{} `json:"token,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

func (h *Handler) CreateDriveOAuthFlow(c *gin.Context) {
	var req createOAuthFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id is required"})
		return
	}

	verifier, challenge, err := newPKCE()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize OAuth"})
		return
	}

	sessionIDStr := ""
	if sessionIDValue, ok := c.Get("session_id"); ok {
		if sessionID, ok := sessionIDValue.(uuid.UUID); ok {
			sessionIDStr = sessionID.String()
		}
	}

	flowID := uuid.NewString()
	flow := oauthFlow{
		ID:            flowID,
		Provider:      oauthProviderDrive,
		SessionID:     sessionIDStr,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(oauthFlowTTL),
		ClientID:      clientID,
		ClientSecret:  strings.TrimSpace(req.ClientSecret),
		Scope:         strings.TrimSpace(req.Scope),
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
	}
	h.oauthFlows.put(flow)

	base := requestBaseURL(c.Request)
	startURL := fmt.Sprintf("%s/api/v1/oauth/drive/start?flow_id=%s", base, url.QueryEscape(flowID))

	c.JSON(http.StatusOK, createOAuthFlowResponse{
		FlowID:    flowID,
		StartURL:  startURL,
		ExpiresAt: flow.ExpiresAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) CreateOneDriveOAuthFlow(c *gin.Context) {
	var req createOAuthFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id is required"})
		return
	}

	verifier, challenge, err := newPKCE()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize OAuth"})
		return
	}

	sessionIDStr := ""
	if sessionIDValue, ok := c.Get("session_id"); ok {
		if sessionID, ok := sessionIDValue.(uuid.UUID); ok {
			sessionIDStr = sessionID.String()
		}
	}

	region := strings.TrimSpace(req.Region)
	if region == "" {
		region = "global"
	}

	flowID := uuid.NewString()
	flow := oauthFlow{
		ID:            flowID,
		Provider:      oauthProviderOneDrive,
		SessionID:     sessionIDStr,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(oauthFlowTTL),
		ClientID:      clientID,
		ClientSecret:  strings.TrimSpace(req.ClientSecret),
		Region:        region,
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
	}
	h.oauthFlows.put(flow)

	base := requestBaseURL(c.Request)
	startURL := fmt.Sprintf("%s/api/v1/oauth/onedrive/start?flow_id=%s", base, url.QueryEscape(flowID))

	c.JSON(http.StatusOK, createOAuthFlowResponse{
		FlowID:    flowID,
		StartURL:  startURL,
		ExpiresAt: flow.ExpiresAt.UTC().Format(time.RFC3339Nano),
	})
}

func (h *Handler) GetDriveOAuthFlow(c *gin.Context) {
	flowID := strings.TrimSpace(c.Param("flowId"))
	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flowId is required"})
		return
	}

	flow, ok := h.oauthFlows.get(flowID)
	if !ok || flow.Provider != oauthProviderDrive {
		c.JSON(http.StatusNotFound, gin.H{"error": "OAuth flow not found or expired"})
		return
	}

	sessionIDStr := ""
	if sessionIDValue, ok := c.Get("session_id"); ok {
		if sessionID, ok := sessionIDValue.(uuid.UUID); ok {
			sessionIDStr = sessionID.String()
		}
	}
	if flow.SessionID != "" && flow.SessionID != sessionIDStr {
		c.JSON(http.StatusNotFound, gin.H{"error": "OAuth flow not found or expired"})
		return
	}

	if !flow.Completed {
		c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "pending"})
		return
	}

	defer h.oauthFlows.delete(flowID)

	if strings.TrimSpace(flow.ResultError) != "" {
		c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "error", Error: flow.ResultError})
		return
	}

	if strings.TrimSpace(flow.ResultTokenJSON) == "" {
		c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "error", Error: "missing token"})
		return
	}

	var token map[string]interface{}
	if err := json.Unmarshal([]byte(flow.ResultTokenJSON), &token); err != nil {
		c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "error", Error: "token decode failed"})
		return
	}

	c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "success", Token: token})
}

func (h *Handler) GetOneDriveOAuthFlow(c *gin.Context) {
	flowID := strings.TrimSpace(c.Param("flowId"))
	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flowId is required"})
		return
	}

	flow, ok := h.oauthFlows.get(flowID)
	if !ok || flow.Provider != oauthProviderOneDrive {
		c.JSON(http.StatusNotFound, gin.H{"error": "OAuth flow not found or expired"})
		return
	}

	sessionIDStr := ""
	if sessionIDValue, ok := c.Get("session_id"); ok {
		if sessionID, ok := sessionIDValue.(uuid.UUID); ok {
			sessionIDStr = sessionID.String()
		}
	}
	if flow.SessionID != "" && flow.SessionID != sessionIDStr {
		c.JSON(http.StatusNotFound, gin.H{"error": "OAuth flow not found or expired"})
		return
	}

	if !flow.Completed {
		c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "pending"})
		return
	}

	defer h.oauthFlows.delete(flowID)

	if strings.TrimSpace(flow.ResultError) != "" {
		c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "error", Error: flow.ResultError})
		return
	}

	if strings.TrimSpace(flow.ResultTokenJSON) == "" {
		c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "error", Error: "missing token"})
		return
	}

	var token map[string]interface{}
	if err := json.Unmarshal([]byte(flow.ResultTokenJSON), &token); err != nil {
		c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "error", Error: "token decode failed"})
		return
	}

	c.JSON(http.StatusOK, oauthFlowStatusResponse{Status: "success", Token: token})
}

func (h *Handler) OAuthDriveStart(c *gin.Context) {
	flowID := strings.TrimSpace(c.Query("flow_id"))
	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow_id is required"})
		return
	}

	flow, ok := h.oauthFlows.get(flowID)
	if !ok || flow.Provider != oauthProviderDrive {
		c.JSON(http.StatusNotFound, gin.H{"error": "OAuth flow not found or expired"})
		return
	}
	if flow.Completed {
		c.JSON(http.StatusConflict, gin.H{"error": "OAuth flow already completed"})
		return
	}

	base := requestBaseURL(c.Request)
	redirectURI := base + "/api/v1/oauth/drive/callback"

	scopes := googleDriveScopes(flow.Scope)
	if scopes == "" {
		scopes = "https://www.googleapis.com/auth/drive"
	}

	authURL, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	query := authURL.Query()
	query.Set("client_id", flow.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("scope", scopes)
	query.Set("access_type", "offline")
	query.Set("prompt", "consent")
	query.Set("include_granted_scopes", "true")
	query.Set("state", flow.ID)
	query.Set("code_challenge", flow.CodeChallenge)
	query.Set("code_challenge_method", "S256")
	authURL.RawQuery = query.Encode()

	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Redirect(http.StatusFound, authURL.String())
}

func (h *Handler) OAuthOneDriveStart(c *gin.Context) {
	flowID := strings.TrimSpace(c.Query("flow_id"))
	if flowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "flow_id is required"})
		return
	}

	flow, ok := h.oauthFlows.get(flowID)
	if !ok || flow.Provider != oauthProviderOneDrive {
		c.JSON(http.StatusNotFound, gin.H{"error": "OAuth flow not found or expired"})
		return
	}
	if flow.Completed {
		c.JSON(http.StatusConflict, gin.H{"error": "OAuth flow already completed"})
		return
	}

	base := requestBaseURL(c.Request)
	redirectURI := base + "/api/v1/oauth/onedrive/callback"

	authority := oneDriveAuthorityBase(flow.Region)
	authURL, _ := url.Parse(authority + "/common/oauth2/v2.0/authorize")
	query := authURL.Query()
	query.Set("client_id", flow.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("response_mode", "query")
	query.Set("scope", "offline_access Files.ReadWrite.All")
	query.Set("state", flow.ID)
	query.Set("code_challenge", flow.CodeChallenge)
	query.Set("code_challenge_method", "S256")
	authURL.RawQuery = query.Encode()

	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Redirect(http.StatusFound, authURL.String())
}

func (h *Handler) OAuthDriveCallback(c *gin.Context) {
	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	oauthErr := strings.TrimSpace(c.Query("error"))
	oauthErrDesc := strings.TrimSpace(c.Query("error_description"))

	flow, ok := h.oauthFlows.get(state)
	if state == "" || !ok || flow.Provider != oauthProviderDrive {
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "drive",
			FlowID:   state,
			OK:       false,
			Error:    "OAuth flow expired or invalid",
		})
		return
	}
	if flow.Completed {
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "drive",
			FlowID:   state,
			OK:       false,
			Error:    "OAuth flow already completed",
		})
		return
	}

	if oauthErr != "" {
		h.oauthFlows.setResult(state, "", strings.TrimSpace(oauthErr+": "+oauthErrDesc))
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "drive",
			FlowID:   state,
			OK:       false,
			Error:    strings.TrimSpace(oauthErr + ": " + oauthErrDesc),
		})
		return
	}

	if code == "" {
		h.oauthFlows.setResult(state, "", "Missing authorization code")
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "drive",
			FlowID:   state,
			OK:       false,
			Error:    "Missing authorization code",
		})
		return
	}

	base := requestBaseURL(c.Request)
	redirectURI := base + "/api/v1/oauth/drive/callback"

	token, err := exchangeGoogleDriveToken(c.Request.Context(), flow, code, redirectURI)
	if err != nil {
		h.oauthFlows.setResult(state, "", err.Error())
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "drive",
			FlowID:   state,
			OK:       false,
			Error:    err.Error(),
		})
		return
	}

	tokenBytes, err := json.Marshal(token)
	if err != nil {
		h.oauthFlows.setResult(state, "", "Failed to encode token")
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "drive",
			FlowID:   state,
			OK:       false,
			Error:    "Failed to encode token",
		})
		return
	}
	h.oauthFlows.setResult(state, string(tokenBytes), "")
	renderOAuthPopup(c, oauthPopupMessage{
		Type:     "rclone-oauth-result",
		Provider: "drive",
		FlowID:   state,
		OK:       true,
	})
}

func (h *Handler) OAuthOneDriveCallback(c *gin.Context) {
	state := strings.TrimSpace(c.Query("state"))
	code := strings.TrimSpace(c.Query("code"))
	oauthErr := strings.TrimSpace(c.Query("error"))
	oauthErrDesc := strings.TrimSpace(c.Query("error_description"))

	flow, ok := h.oauthFlows.get(state)
	if state == "" || !ok || flow.Provider != oauthProviderOneDrive {
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "onedrive",
			FlowID:   state,
			OK:       false,
			Error:    "OAuth flow expired or invalid",
		})
		return
	}
	if flow.Completed {
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "onedrive",
			FlowID:   state,
			OK:       false,
			Error:    "OAuth flow already completed",
		})
		return
	}

	if oauthErr != "" {
		h.oauthFlows.setResult(state, "", strings.TrimSpace(oauthErr+": "+oauthErrDesc))
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "onedrive",
			FlowID:   state,
			OK:       false,
			Error:    strings.TrimSpace(oauthErr + ": " + oauthErrDesc),
		})
		return
	}

	if code == "" {
		h.oauthFlows.setResult(state, "", "Missing authorization code")
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "onedrive",
			FlowID:   state,
			OK:       false,
			Error:    "Missing authorization code",
		})
		return
	}

	base := requestBaseURL(c.Request)
	redirectURI := base + "/api/v1/oauth/onedrive/callback"

	token, err := exchangeOneDriveToken(c.Request.Context(), flow, code, redirectURI)
	if err != nil {
		h.oauthFlows.setResult(state, "", err.Error())
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "onedrive",
			FlowID:   state,
			OK:       false,
			Error:    err.Error(),
		})
		return
	}

	tokenBytes, err := json.Marshal(token)
	if err != nil {
		h.oauthFlows.setResult(state, "", "Failed to encode token")
		renderOAuthPopup(c, oauthPopupMessage{
			Type:     "rclone-oauth-result",
			Provider: "onedrive",
			FlowID:   state,
			OK:       false,
			Error:    "Failed to encode token",
		})
		return
	}
	h.oauthFlows.setResult(state, string(tokenBytes), "")
	renderOAuthPopup(c, oauthPopupMessage{
		Type:     "rclone-oauth-result",
		Provider: "onedrive",
		FlowID:   state,
		OK:       true,
	})
}

type oauthPopupMessage struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	FlowID   string `json:"flow_id"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
}

func renderOAuthPopup(c *gin.Context, payload oauthPopupMessage) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Frame-Options", "DENY")

	body := "授权完成，可关闭此窗口。"
	if !payload.OK {
		body = "授权失败：" + payload.Error
	}

	jsonBytes, _ := json.Marshal(payload)
	jsonBytes = bytes.ReplaceAll(jsonBytes, []byte("</"), []byte("<\\/"))

	page := fmt.Sprintf(`<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <meta name="referrer" content="no-referrer" />
    <title>OAuth</title>
  </head>
  <body>
    <script>
      (function () {
        var payload = %s;
        try {
          if (window.opener && window.opener.postMessage) {
            window.opener.postMessage(payload, "*");
          }
        } catch (e) {}
        try { window.close(); } catch (e) {}
      })();
    </script>
    <p>%s</p>
  </body>
</html>`, string(jsonBytes), html.EscapeString(body))

	c.String(http.StatusOK, page)
}

func requestBaseURL(r *http.Request) string {
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	} else if idx := strings.Index(proto, ","); idx >= 0 {
		proto = strings.TrimSpace(proto[:idx])
	}

	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	} else if idx := strings.Index(host, ","); idx >= 0 {
		host = strings.TrimSpace(host[:idx])
	}

	return proto + "://" + host
}

func googleDriveScopes(scope string) string {
	trimmed := strings.TrimSpace(scope)
	if trimmed == "" {
		return ""
	}

	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ' ' || r == ',' || r == ';'
	})

	var scopes []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		scopes = append(scopes, googleDriveScopeToURL(part))
	}

	return strings.Join(scopes, " ")
}

func googleDriveScopeToURL(scope string) string {
	if strings.HasPrefix(scope, "http://") || strings.HasPrefix(scope, "https://") {
		return scope
	}

	switch scope {
	case "drive":
		return "https://www.googleapis.com/auth/drive"
	case "drive.readonly":
		return "https://www.googleapis.com/auth/drive.readonly"
	case "drive.file":
		return "https://www.googleapis.com/auth/drive.file"
	case "drive.appfolder":
		return "https://www.googleapis.com/auth/drive.appfolder"
	case "drive.metadata.readonly":
		return "https://www.googleapis.com/auth/drive.metadata.readonly"
	default:
		return "https://www.googleapis.com/auth/" + scope
	}
}

func oneDriveAuthorityBase(region string) string {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case "us":
		return "https://login.microsoftonline.us"
	case "de":
		return "https://login.microsoftonline.de"
	case "cn":
		return "https://login.chinacloudapi.cn"
	default:
		return "https://login.microsoftonline.com"
	}
}

type googleTokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int64  `json:"expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type microsoftTokenResponse struct {
	TokenType        string `json:"token_type"`
	ExpiresIn        int64  `json:"expires_in"`
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func exchangeGoogleDriveToken(ctx context.Context, flow oauthFlow, code string, redirectURI string) (map[string]interface{}, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", flow.ClientID)
	if strings.TrimSpace(flow.ClientSecret) != "" {
		form.Set("client_secret", flow.ClientSecret)
	}
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", flow.CodeVerifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	var parsed googleTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("OAuth token response decode failed")
	}

	if resp.StatusCode != http.StatusOK || parsed.Error != "" {
		msg := strings.TrimSpace(parsed.Error + ": " + parsed.ErrorDescription)
		if msg == ":" || msg == "" {
			msg = "OAuth token exchange failed (HTTP " + strconv.Itoa(resp.StatusCode) + ")"
		}
		return nil, fmt.Errorf(msg)
	}

	expiry := time.Now().UTC().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	return map[string]interface{}{
		"access_token":  parsed.AccessToken,
		"token_type":    parsed.TokenType,
		"refresh_token": parsed.RefreshToken,
		"expiry":        expiry.Format(time.RFC3339Nano),
	}, nil
}

func exchangeOneDriveToken(ctx context.Context, flow oauthFlow, code string, redirectURI string) (map[string]interface{}, error) {
	form := url.Values{}
	form.Set("client_id", flow.ClientID)
	if strings.TrimSpace(flow.ClientSecret) != "" {
		form.Set("client_secret", flow.ClientSecret)
	}
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", flow.CodeVerifier)
	form.Set("scope", "offline_access Files.ReadWrite.All")

	tokenURL := oneDriveAuthorityBase(flow.Region) + "/common/oauth2/v2.0/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	var parsed microsoftTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("OAuth token response decode failed")
	}

	if resp.StatusCode != http.StatusOK || parsed.Error != "" {
		msg := strings.TrimSpace(parsed.Error + ": " + parsed.ErrorDescription)
		if msg == ":" || msg == "" {
			msg = "OAuth token exchange failed (HTTP " + strconv.Itoa(resp.StatusCode) + ")"
		}
		return nil, fmt.Errorf(msg)
	}

	expiry := time.Now().UTC().Add(time.Duration(parsed.ExpiresIn) * time.Second)
	return map[string]interface{}{
		"access_token":  parsed.AccessToken,
		"token_type":    parsed.TokenType,
		"refresh_token": parsed.RefreshToken,
		"expiry":        expiry.Format(time.RFC3339Nano),
	}, nil
}
