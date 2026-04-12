package tds

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Client communicates with the TOTVS Language Server (advpls) via JSON-RPC 2.0 over stdio.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	mu     sync.Mutex
	nextID atomic.Int64

	// Connection state
	connectionToken string
	environment     string
}

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// FindAdvplsBinary locates the advpls binary in standard locations.
func FindAdvplsBinary() (string, error) {
	// 1. Check PATH
	if path, err := exec.LookPath("advpls"); err == nil {
		return path, nil
	}

	// 2. Check ~/.config/mapj/bin/
	home, _ := os.UserHomeDir()
	if home != "" {
		binName := "advpls"
		if runtime.GOOS == "windows" {
			binName = "advpls.exe"
		}
		candidate := filepath.Join(home, ".config", "mapj", "bin", binName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// 3. Check npm global install
	if path, err := exec.LookPath("advpls"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("advpls binary not found. Install via: npm install -g @totvs/tds-ls, or place it in PATH or ~/.config/mapj/bin/")
}

// NewClient creates a new TDS Language Server client.
// It spawns the advpls process in language-server mode.
func NewClient(ctx context.Context, advplsPath string) (*Client, error) {
	cmd := exec.CommandContext(ctx, advplsPath, "language-server", "--fs-encoding", "CP1252")
	cmd.Stderr = io.Discard

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start advpls: %w", err)
	}

	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		reader: bufio.NewReader(stdout),
	}

	// Send LSP initialize
	if err := c.initialize(ctx); err != nil {
		c.Close()
		return nil, fmt.Errorf("LSP initialize failed: %w", err)
	}

	return c, nil
}

// Close shuts down the language server and waits for the process to exit.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Send shutdown request (best effort)
	_ = c.sendRaw("shutdown", nil)
	_ = c.sendNotification("exit", nil)

	c.stdin.Close()
	return c.cmd.Wait()
}

// ConnectionToken returns the current connection token.
func (c *Client) ConnectionToken() string {
	return c.connectionToken
}

// Environment returns the current environment.
func (c *Client) Environment() string {
	return c.environment
}

// ─── LSP lifecycle ──────────────────────────────────────────────────────────

func (c *Client) initialize(ctx context.Context) error {
	params := map[string]any{
		"processId": os.Getpid(),
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"completion": map[string]any{},
			},
		},
		"rootUri": "file:///",
	}

	var result json.RawMessage
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return err
	}

	// Send initialized notification
	return c.sendNotification("initialized", map[string]any{})
}

// ─── Server operations ──────────────────────────────────────────────────────

// Validate retrieves build info from an AppServer without authenticating.
func (c *Client) Validate(ctx context.Context, server string, port int) (*ValidationResult, error) {
	params := ValidationParams{}
	params.ValidationInfo.Server = server
	params.ValidationInfo.Port = port
	params.ValidationInfo.ServerType = "totvs_server_protheus"

	var result ValidationResult
	if err := c.call(ctx, "$totvsserver/validation", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Connect establishes a connection to the AppServer.
func (c *Client) Connect(ctx context.Context, server string, port int, build string, secure bool) (*ConnectResult, error) {
	params := ConnectParams{}
	p := &params.ConnectionInfo
	p.ConnType = 1 // standard
	p.ServerName = fmt.Sprintf("%s:%d", server, port)
	p.Identification = p.ServerName
	p.ServerType = 1 // Protheus
	p.Server = server
	p.Port = port
	p.Build = build
	if secure {
		p.BSecure = 1
	}
	p.Environment = ""
	p.AutoReconnect = true

	var result ConnectResult
	if err := c.call(ctx, "$totvsserver/connect", params, &result); err != nil {
		return nil, err
	}

	if result.ConnectionToken != "" {
		c.connectionToken = result.ConnectionToken
	}

	return &result, nil
}

// Authenticate authenticates with the connected AppServer.
func (c *Client) Authenticate(ctx context.Context, environment, user, password string) (*AuthResult, error) {
	params := AuthParams{}
	p := &params.AuthenticationInfo
	p.ConnectionToken = c.connectionToken
	p.Environment = environment
	p.User = user
	p.Password = password
	p.Encoding = "CP1252"

	var result AuthResult
	if err := c.call(ctx, "$totvsserver/authentication", params, &result); err != nil {
		return nil, err
	}

	if result.ConnectionToken != "" {
		c.connectionToken = result.ConnectionToken
		c.environment = environment
	}

	return &result, nil
}

// ConnectAndAuth is a convenience method that validates, connects, and authenticates.
func (c *Client) ConnectAndAuth(ctx context.Context, server string, port int, environment, user, password string, secure bool) error {
	// Step 1: Validate to get build version
	valResult, err := c.Validate(ctx, server, port)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Step 2: Connect
	connResult, err := c.Connect(ctx, server, port, valResult.Build, secure)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	if connResult.ConnectionToken == "" {
		return fmt.Errorf("connection failed: no token returned")
	}

	// Step 3: Authenticate
	authResult, err := c.Authenticate(ctx, environment, user, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	if authResult.ConnectionToken == "" {
		return fmt.Errorf("authentication failed: no token returned")
	}

	return nil
}

// Disconnect closes the connection to the AppServer.
func (c *Client) Disconnect(ctx context.Context) error {
	params := DisconnectParams{}
	params.DisconnectInfo.ConnectionToken = c.connectionToken
	params.DisconnectInfo.ServerName = ""

	var result json.RawMessage
	err := c.call(ctx, "$totvsserver/disconnect", params, &result)
	c.connectionToken = ""
	c.environment = ""
	return err
}

// ─── Compilation ────────────────────────────────────────────────────────────

// Compile sends files for compilation on the AppServer.
func (c *Client) Compile(ctx context.Context, files []string, includes []string, authToken string, recompile bool) (*CompileResult, error) {
	params := CompileParams{}
	p := &params.CompilationInfo
	p.ConnectionToken = c.connectionToken
	p.AuthorizationToken = authToken
	p.Environment = c.environment
	p.IncludeUris = includes
	p.FileUris = files
	p.ExtensionsAllowed = []string{".prw", ".prx", ".prg", ".aph", ".tlpp", ".4gl", ".per", ".apl"}
	p.IncludeUrisRequired = true
	if recompile {
		p.CompileOptions = map[string]bool{"recompile": true}
	}

	var result CompileResult
	if err := c.call(ctx, "$totvsserver/compilation", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ─── RPO operations ─────────────────────────────────────────────────────────

// InspectObjects lists all objects in the RPO.
func (c *Client) InspectObjects(ctx context.Context, includeTres bool) ([]ObjectData, error) {
	params := InspectorObjectsParams{}
	params.InspectorObjectsInfo.ConnectionToken = c.connectionToken
	params.InspectorObjectsInfo.Environment = c.environment
	params.InspectorObjectsInfo.IncludeTres = includeTres

	var result InspectorObjectsResult
	if err := c.call(ctx, "$totvsserver/inspectorObjects", params, &result); err != nil {
		return nil, err
	}

	return parseObjects(result.Objects), nil
}

// InspectFunctions lists all functions in the RPO.
func (c *Client) InspectFunctions(ctx context.Context) ([]FunctionData, error) {
	params := InspectorFunctionsParams{}
	params.InspectorFunctionsInfo.ConnectionToken = c.connectionToken
	params.InspectorFunctionsInfo.Environment = c.environment

	var result InspectorFunctionsResult
	if err := c.call(ctx, "$totvsserver/inspectorFunctions", params, &result); err != nil {
		return nil, err
	}

	return parseFunctions(result.Functions), nil
}

// RpoInfo gets RPO information.
func (c *Client) RpoInfo(ctx context.Context) (json.RawMessage, error) {
	params := RpoInfoParams{}
	params.RpoInfo.ConnectionToken = c.connectionToken
	params.RpoInfo.Environment = c.environment

	var result json.RawMessage
	if err := c.call(ctx, "$totvsserver/rpoInfo", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RpoCheckIntegrity checks the RPO integrity.
func (c *Client) RpoCheckIntegrity(ctx context.Context) (*RpoCheckIntegrityResult, error) {
	params := RpoCheckIntegrityParams{}
	params.RpoCheckIntegrityInfo.ConnectionToken = c.connectionToken
	params.RpoCheckIntegrityInfo.Environment = c.environment

	var result RpoCheckIntegrityResult
	if err := c.call(ctx, "$totvsserver/rpoCheckIntegrity", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DefragRpo defragments the RPO.
func (c *Client) DefragRpo(ctx context.Context, authToken string, cleanHistory bool) error {
	params := DefragParams{}
	p := &params.DefragRpoInfo
	p.ConnectionToken = c.connectionToken
	p.AuthorizationToken = authToken
	p.Environment = c.environment
	p.PackPatchInfo = cleanHistory

	var result json.RawMessage
	return c.call(ctx, "$totvsserver/defragRpo", params, &result)
}

// RevalidateRpo revalidates the RPO.
func (c *Client) RevalidateRpo(ctx context.Context) error {
	params := RevalidateRpoParams{}
	params.RevalidateRpoInfo.ConnectionToken = c.connectionToken
	params.RevalidateRpoInfo.Environment = c.environment

	var result json.RawMessage
	return c.call(ctx, "$totvsserver/revalidateRpo", params, &result)
}

// DeletePrograms removes programs from the RPO.
func (c *Client) DeletePrograms(ctx context.Context, programs []string, authToken string) (*DeleteProgramsResult, error) {
	params := DeleteProgramsParams{}
	p := &params.DeleteProgramsInfo
	p.ConnectionToken = c.connectionToken
	p.AuthorizationToken = authToken
	p.Environment = c.environment
	p.Programs = programs

	var result DeleteProgramsResult
	if err := c.call(ctx, "$totvsserver/deletePrograms", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ─── Patch operations ───────────────────────────────────────────────────────

// PatchGenerate generates a patch from RPO resources.
func (c *Client) PatchGenerate(ctx context.Context, resources []string, patchType int, saveLocal string, authToken string, patchName string) error {
	params := PatchGenerateParams{}
	p := &params.PatchGenerateInfo
	p.ConnectionToken = c.connectionToken
	p.AuthorizationToken = authToken
	p.Environment = c.environment
	p.IsLocal = true
	p.PatchType = patchType
	p.SaveLocal = saveLocal
	p.FileResources = resources
	p.PatchName = patchName

	var result json.RawMessage
	return c.call(ctx, "$totvsserver/patchGenerate", params, &result)
}

// PatchApply applies a patch to the RPO.
func (c *Client) PatchApply(ctx context.Context, patchUri string, isLocal bool, authToken string) error {
	params := PatchApplyParams{}
	p := &params.PatchApplyInfo
	p.ConnectionToken = c.connectionToken
	p.AuthorizationToken = authToken
	p.Environment = c.environment
	p.PatchUri = patchUri
	p.IsLocal = isLocal

	var result json.RawMessage
	return c.call(ctx, "$totvsserver/patchApply", params, &result)
}

// PatchInfo returns information about a patch file.
func (c *Client) PatchInfo(ctx context.Context, patchUri string, authToken string) ([]PatchInfoEntry, error) {
	params := PatchInfoParams{}
	p := &params.PatchInfoInfo
	p.ConnectionToken = c.connectionToken
	p.AuthorizationToken = authToken
	p.Environment = c.environment
	p.PatchUri = patchUri
	p.IsLocal = true

	var result struct {
		PatchInfos []PatchInfoEntry `json:"patchInfos"`
	}
	if err := c.call(ctx, "$totvsserver/patchInfo", params, &result); err != nil {
		return nil, err
	}
	return result.PatchInfos, nil
}

// ─── Monitor ────────────────────────────────────────────────────────────────

// GetUsers returns the list of connected users.
func (c *Client) GetUsers(ctx context.Context) ([]MonitorUser, error) {
	params := GetUsersParams{}
	params.GetUsersInfo.ConnectionToken = c.connectionToken

	var result struct {
		MntUsers []MonitorUser `json:"mntUsers"`
	}
	if err := c.call(ctx, "$totvsmonitor/getUsers", params, &result); err != nil {
		return nil, err
	}
	return result.MntUsers, nil
}

// KillUser terminates a user connection.
func (c *Client) KillUser(ctx context.Context, username, computerName string, threadID int, serverName string) (string, error) {
	params := KillUserParams{}
	p := &params.KillUserInfo
	p.ConnectionToken = c.connectionToken
	p.UserName = username
	p.ComputerName = computerName
	p.ThreadID = threadID
	p.ServerName = serverName

	var result struct {
		Message string `json:"message"`
	}
	if err := c.call(ctx, "$totvsmonitor/killUser", params, &result); err != nil {
		return "", err
	}
	return result.Message, nil
}

// SetConnectionStatus locks or unlocks server connections.
func (c *Client) SetConnectionStatus(ctx context.Context, allowConnections bool) error {
	params := SetConnectionStatusParams{}
	params.SetConnectionStatusInfo.ConnectionToken = c.connectionToken
	params.SetConnectionStatusInfo.Status = allowConnections

	var result json.RawMessage
	return c.call(ctx, "$totvsmonitor/setConnectionStatus", params, &result)
}

// ─── JSON-RPC transport ─────────────────────────────────────────────────────

func (c *Client) call(ctx context.Context, method string, params any, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID.Add(1)

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.writeMessage(req); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	// Read responses until we get one matching our ID
	for {
		resp, err := c.readMessage()
		if err != nil {
			return fmt.Errorf("read response: %w", err)
		}

		if resp.ID == id {
			if resp.Error != nil {
				return fmt.Errorf("LSP error %d: %s", resp.Error.Code, resp.Error.Message)
			}
			if result != nil && resp.Result != nil {
				return json.Unmarshal(resp.Result, result)
			}
			return nil
		}
		// Skip notifications and responses for other IDs
	}
}

func (c *Client) sendRaw(method string, params any) error {
	id := c.nextID.Add(1)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	return c.writeMessage(req)
}

func (c *Client) sendNotification(method string, params any) error {
	type notification struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}
	n := notification{JSONRPC: "2.0", Method: method, Params: params}
	return c.writeMessage(n)
}

func (c *Client) writeMessage(msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	_, err = c.stdin.Write(body)
	return err
}

func (c *Client) readMessage() (*jsonRPCResponse, error) {
	// Read headers
	contentLength := 0
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break // End of headers
		}
		if strings.HasPrefix(line, "Content-Length:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			contentLength, err = strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("invalid Content-Length: %s", val)
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	// Read body
	body := make([]byte, contentLength)
	if _, err := io.ReadFull(c.reader, body); err != nil {
		return nil, err
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

// ─── Parsers (match tds-vscode regex patterns) ──────────────────────────────

var objectRegex = regexp.MustCompile(`(.*)\s\((.*)\)\s(.)(.?)`)

func parseObjects(raw []string) []ObjectData {
	result := make([]ObjectData, 0, len(raw))
	for _, line := range raw {
		groups := objectRegex.FindStringSubmatch(line)
		if groups != nil && len(groups) >= 4 {
			d := ObjectData{
				Source: strings.TrimSpace(groups[1]),
				Date:   groups[2],
			}
			if len(groups) > 3 {
				d.SourceStatus = groups[3]
			}
			if len(groups) > 4 {
				d.RpoStatus = groups[4]
			}
			result = append(result, d)
		} else {
			result = append(result, ObjectData{Source: line})
		}
	}
	return result
}

var functionRegex = regexp.MustCompile(`(#NONE#)?(.*?)(?:#NONE#)?\s\((.*):(\d+)\)\s?(.)(.?)`)

func parseFunctions(raw []string) []FunctionData {
	result := make([]FunctionData, 0, len(raw))
	for _, line := range raw {
		groups := functionRegex.FindStringSubmatch(line)
		if groups != nil && len(groups) >= 5 {
			lineNum, _ := strconv.Atoi(groups[4])
			funcName := groups[2]
			if groups[1] != "" { // #NONE# prefix = private
				funcName = "#" + strings.TrimSuffix(funcName, "#")
			}
			d := FunctionData{
				Function: funcName,
				Source:   groups[3],
				Line:     lineNum,
			}
			if len(groups) > 5 {
				d.SourceStatus = groups[5]
			}
			if len(groups) > 6 {
				d.RpoStatus = groups[6]
			}
			result = append(result, d)
		} else {
			result = append(result, FunctionData{Function: line})
		}
	}
	return result
}
