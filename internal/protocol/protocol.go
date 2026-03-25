package protocol

import "encoding/json"

// InfoResponse is the expected JSON output of the "info" subcommand.
type InfoResponse struct {
	ProtocolVersion int          `json:"protocol_version"`
	Name            string       `json:"name"`
	Type            string       `json:"type"`
	Description     string       `json:"description"`
	IsPreview       bool         `json:"is_preview"`
	ProtectedDirs   []string     `json:"protected_dirs"`
	HookNames       []string     `json:"hook_names"`
	Capabilities    Capabilities `json:"capabilities"`
}

// Capabilities declares which optional interfaces the agent implements.
type Capabilities struct {
	Hooks                  bool `json:"hooks"`
	TranscriptAnalyzer     bool `json:"transcript_analyzer"`
	TranscriptPreparer     bool `json:"transcript_preparer"`
	TokenCalculator        bool `json:"token_calculator"`
	TextGenerator          bool `json:"text_generator"`
	HookResponseWriter     bool `json:"hook_response_writer"`
	SubagentAwareExtractor bool `json:"subagent_aware_extractor"`
	UsesTerminal           bool `json:"uses_terminal"`
}

// DetectResponse is the output of the "detect" subcommand.
type DetectResponse struct {
	Present bool `json:"present"`
}

// SessionIDResponse is the output of the "get-session-id" subcommand.
type SessionIDResponse struct {
	SessionID string `json:"session_id"`
}

// SessionDirResponse is the output of the "get-session-dir" subcommand.
type SessionDirResponse struct {
	SessionDir string `json:"session_dir"`
}

// SessionFileResponse is the output of the "resolve-session-file" subcommand.
type SessionFileResponse struct {
	SessionFile string `json:"session_file"`
}

// ChunkResponse is the output of "chunk-transcript" and the input to "reassemble-transcript".
type ChunkResponse struct {
	Chunks [][]byte `json:"chunks"`
}

// AgentSessionJSON is the session data format for "write-session" and "read-session".
type AgentSessionJSON struct {
	SessionID     string   `json:"session_id"`
	AgentName     string   `json:"agent_name"`
	RepoPath      string   `json:"repo_path"`
	SessionRef    string   `json:"session_ref"`
	StartTime     string   `json:"start_time"`
	NativeData    []byte   `json:"native_data"`
	ModifiedFiles []string `json:"modified_files"`
	NewFiles      []string `json:"new_files"`
	DeletedFiles  []string `json:"deleted_files"`
}

// EventJSON is the output of "parse-hook" when the hook has lifecycle significance.
type EventJSON struct {
	Type              int               `json:"type"`
	SessionID         string            `json:"session_id"`
	PreviousSessionID string            `json:"previous_session_id,omitempty"`
	SessionRef        string            `json:"session_ref,omitempty"`
	Prompt            string            `json:"prompt,omitempty"`
	Model             string            `json:"model,omitempty"`
	Timestamp         string            `json:"timestamp,omitempty"`
	ToolUseID         string            `json:"tool_use_id,omitempty"`
	SubagentID        string            `json:"subagent_id,omitempty"`
	ToolInput         json.RawMessage   `json:"tool_input,omitempty"`
	SubagentType      string            `json:"subagent_type,omitempty"`
	TaskDescription   string            `json:"task_description,omitempty"`
	ResponseMessage   string            `json:"response_message,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// HooksInstalledResponse is the output of "install-hooks".
type HooksInstalledResponse struct {
	HooksInstalled int `json:"hooks_installed"`
}

// AreHooksInstalledResponse is the output of "are-hooks-installed".
type AreHooksInstalledResponse struct {
	Installed bool `json:"installed"`
}

// ResumeCommandResponse is the output of "format-resume-command".
type ResumeCommandResponse struct {
	Command string `json:"command"`
}

// TranscriptPositionResponse is the output of "get-transcript-position".
type TranscriptPositionResponse struct {
	Position int `json:"position"`
}

// ExtractFilesResponse is the output of "extract-modified-files".
type ExtractFilesResponse struct {
	Files           []string `json:"files"`
	CurrentPosition int      `json:"current_position"`
}

// ExtractPromptsResponse is the output of "extract-prompts".
type ExtractPromptsResponse struct {
	Prompts []string `json:"prompts"`
}

// ExtractSummaryResponse is the output of "extract-summary".
type ExtractSummaryResponse struct {
	Summary    string `json:"summary"`
	HasSummary bool   `json:"has_summary"`
}

// TokenUsageResponse is the output of token calculation subcommands.
type TokenUsageResponse struct {
	InputTokens         int                `json:"input_tokens"`
	CacheCreationTokens int                `json:"cache_creation_tokens"`
	CacheReadTokens     int                `json:"cache_read_tokens"`
	OutputTokens        int                `json:"output_tokens"`
	APICallCount        int                `json:"api_call_count"`
	SubagentTokens      *TokenUsageResponse `json:"subagent_tokens,omitempty"`
}

// GenerateTextResponse is the output of "generate-text".
type GenerateTextResponse struct {
	Text string `json:"text"`
}

// HookInputJSON is the standard input format for hook-related subcommands.
type HookInputJSON struct {
	HookType   string          `json:"hook_type"`
	SessionID  string          `json:"session_id"`
	SessionRef string          `json:"session_ref"`
	Timestamp  string          `json:"timestamp"`
	UserPrompt string          `json:"user_prompt,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	ToolInput  json.RawMessage `json:"tool_input,omitempty"`
	RawData    json.RawMessage `json:"raw_data,omitempty"`
}
