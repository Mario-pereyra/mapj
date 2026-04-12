// Package tds provides a client for the TOTVS Language Server (advpls)
// using JSON-RPC 2.0 over stdio. It wraps the custom $totvsserver/* protocol
// messages to provide structured operations against a TOTVS Application Server.
//
// Zero CLI imports — this is pure domain logic.
package tds

// ─── Connection ─────────────────────────────────────────────────────────────

// ConnectParams are the parameters for $totvsserver/connect.
type ConnectParams struct {
	ConnectionInfo struct {
		ConnType      int    `json:"connType"`
		ServerName    string `json:"serverName"`
		Identification string `json:"identification"`
		ServerType    int    `json:"serverType"`
		Server        string `json:"server"`
		Port          int    `json:"port"`
		Build         string `json:"build"`
		BSecure       int    `json:"bSecure"`
		Environment   string `json:"environment"`
		AutoReconnect bool   `json:"autoReconnect"`
	} `json:"connectionInfo"`
}

// ConnectResult is returned by $totvsserver/connect.
type ConnectResult struct {
	ID                 interface{} `json:"id"`
	OSType             int         `json:"osType"`
	ConnectionToken    string      `json:"connectionToken"`
	NeedAuthentication bool        `json:"needAuthentication"`
}

// AuthParams are the parameters for $totvsserver/authentication.
type AuthParams struct {
	AuthenticationInfo struct {
		ConnectionToken string `json:"connectionToken"`
		Environment     string `json:"environment"`
		User            string `json:"user"`
		Password        string `json:"password"`
		Encoding        string `json:"encoding"`
	} `json:"authenticationInfo"`
}

// AuthResult is returned by $totvsserver/authentication.
type AuthResult struct {
	ID              interface{} `json:"id"`
	OSType          int         `json:"osType"`
	ConnectionToken string      `json:"connectionToken"`
}

// ValidationParams are the parameters for $totvsserver/validation.
type ValidationParams struct {
	ValidationInfo struct {
		Server     string `json:"server"`
		Port       int    `json:"port"`
		ServerType string `json:"serverType"`
	} `json:"validationInfo"`
}

// ValidationResult is returned by $totvsserver/validation.
type ValidationResult struct {
	Build  string `json:"build"`
	Secure int    `json:"secure"`
}

// DisconnectParams are the parameters for $totvsserver/disconnect.
type DisconnectParams struct {
	DisconnectInfo struct {
		ConnectionToken string `json:"connectionToken"`
		ServerName      string `json:"serverName"`
	} `json:"disconnectInfo"`
}

// ─── Compilation ────────────────────────────────────────────────────────────

// CompileParams are the parameters for $totvsserver/compilation.
type CompileParams struct {
	CompilationInfo struct {
		ConnectionToken     string   `json:"connectionToken"`
		AuthorizationToken  string   `json:"authorizationToken"`
		Environment         string   `json:"environment"`
		IncludeUris         []string `json:"includeUris"`
		FileUris            []string `json:"fileUris"`
		CompileOptions      any      `json:"compileOptions"`
		ExtensionsAllowed   []string `json:"extensionsAllowed"`
		IncludeUrisRequired bool     `json:"includeUrisRequired"`
	} `json:"compilationInfo"`
}

// CompileResult is returned by $totvsserver/compilation.
type CompileResult struct {
	ReturnCode int               `json:"returnCode"`
	Files      []CompileFileInfo `json:"compileInfos"`
}

// CompileFileInfo has per-file compilation results.
type CompileFileInfo struct {
	File   string            `json:"file"`
	Status string            `json:"status"`
	Detail []CompileDiagnostic `json:"detail,omitempty"`
}

// CompileDiagnostic is a single error or warning from compilation.
type CompileDiagnostic struct {
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"` // "error" or "warning"
	Message  string `json:"message"`
}

// ─── RPO ────────────────────────────────────────────────────────────────────

// InspectorObjectsParams are the parameters for $totvsserver/inspectorObjects.
type InspectorObjectsParams struct {
	InspectorObjectsInfo struct {
		ConnectionToken string `json:"connectionToken"`
		Environment     string `json:"environment"`
		IncludeTres     bool   `json:"includeTres"`
	} `json:"inspectorObjectsInfo"`
}

// InspectorObjectsResult is returned by $totvsserver/inspectorObjects.
type InspectorObjectsResult struct {
	Objects []string `json:"objects"`
}

// ObjectData is a parsed RPO object entry.
type ObjectData struct {
	Source       string `json:"source"`
	Date         string `json:"date"`
	RpoStatus    string `json:"rpoStatus"`
	SourceStatus string `json:"sourceStatus"`
}

// InspectorFunctionsParams are the parameters for $totvsserver/inspectorFunctions.
type InspectorFunctionsParams struct {
	InspectorFunctionsInfo struct {
		ConnectionToken string `json:"connectionToken"`
		Environment     string `json:"environment"`
	} `json:"inspectorFunctionsInfo"`
}

// InspectorFunctionsResult is returned by $totvsserver/inspectorFunctions.
type InspectorFunctionsResult struct {
	Functions []string `json:"functions"`
}

// FunctionData is a parsed RPO function entry.
type FunctionData struct {
	Function     string `json:"function"`
	Source       string `json:"source"`
	Line         int    `json:"line"`
	RpoStatus    string `json:"rpoStatus"`
	SourceStatus string `json:"sourceStatus"`
}

// RpoInfoParams are the parameters for $totvsserver/rpoInfo.
type RpoInfoParams struct {
	RpoInfo struct {
		ConnectionToken string `json:"connectionToken"`
		Environment     string `json:"environment"`
	} `json:"rpoInfo"`
}

// DefragParams are the parameters for $totvsserver/defragRpo.
type DefragParams struct {
	DefragRpoInfo struct {
		ConnectionToken    string `json:"connectionToken"`
		AuthorizationToken string `json:"authorizationToken"`
		Environment        string `json:"environment"`
		PackPatchInfo      bool   `json:"packPatchInfo"`
	} `json:"defragRpoInfo"`
}

// RpoCheckIntegrityParams are the parameters for $totvsserver/rpoCheckIntegrity.
type RpoCheckIntegrityParams struct {
	RpoCheckIntegrityInfo struct {
		ConnectionToken string `json:"connectionToken"`
		Environment     string `json:"environment"`
	} `json:"rpoCheckIntegrityInfo"`
}

// RpoCheckIntegrityResult is returned by $totvsserver/rpoCheckIntegrity.
type RpoCheckIntegrityResult struct {
	Integrity bool   `json:"integrity"`
	Message   string `json:"message"`
}

// RevalidateRpoParams are the parameters for $totvsserver/revalidateRpo.
type RevalidateRpoParams struct {
	RevalidateRpoInfo struct {
		ConnectionToken string `json:"connectionToken"`
		Environment     string `json:"environment"`
	} `json:"revalidateRpoInfo"`
}

// DeleteProgramsParams are the parameters for $totvsserver/deletePrograms.
type DeleteProgramsParams struct {
	DeleteProgramsInfo struct {
		ConnectionToken    string   `json:"connectionToken"`
		AuthorizationToken string   `json:"authorizationToken"`
		Environment        string   `json:"environment"`
		Programs           []string `json:"programs"`
	} `json:"deleteProgramsInfo"`
}

// DeleteProgramsResult is returned by $totvsserver/deletePrograms.
type DeleteProgramsResult struct {
	ReturnCode int `json:"returnCode"`
}

// ─── Patch ──────────────────────────────────────────────────────────────────

// PatchGenerateParams are the parameters for $totvsserver/patchGenerate.
type PatchGenerateParams struct {
	PatchGenerateInfo struct {
		ConnectionToken    string   `json:"connectionToken"`
		AuthorizationToken string   `json:"authorizationToken"`
		Environment        string   `json:"environment"`
		IsLocal            bool     `json:"isLocal"`
		PatchType          int      `json:"patchType"` // 1=PTM, 2=UPD, 3=PAK
		SaveLocal          string   `json:"saveLocal"`
		FileResources      []string `json:"fileResources"`
		PatchName          string   `json:"patchName,omitempty"`
	} `json:"patchGenerateInfo"`
}

// PatchApplyParams are the parameters for $totvsserver/patchApply.
type PatchApplyParams struct {
	PatchApplyInfo struct {
		ConnectionToken    string `json:"connectionToken"`
		AuthorizationToken string `json:"authorizationToken"`
		Environment        string `json:"environment"`
		PatchUri           string `json:"patchUri"`
		IsLocal            bool   `json:"isLocal"`
	} `json:"patchApplyInfo"`
}

// PatchInfoParams are the parameters for $totvsserver/patchInfo.
type PatchInfoParams struct {
	PatchInfoInfo struct {
		ConnectionToken    string `json:"connectionToken"`
		AuthorizationToken string `json:"authorizationToken"`
		Environment        string `json:"environment"`
		PatchUri           string `json:"patchUri"`
		IsLocal            bool   `json:"isLocal"`
	} `json:"patchInfoInfo"`
}

// PatchInfoEntry is a single entry from patch info results.
type PatchInfoEntry struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

// ─── Monitor ────────────────────────────────────────────────────────────────

// GetUsersParams are the parameters for $totvsmonitor/getUsers.
type GetUsersParams struct {
	GetUsersInfo struct {
		ConnectionToken string `json:"connectionToken"`
	} `json:"getUsersInfo"`
}

// MonitorUser represents a connected user from the monitor.
type MonitorUser struct {
	Username     string `json:"username"`
	ComputerName string `json:"computerName"`
	ThreadID     int    `json:"threadId"`
	Server       string `json:"server"`
	MainName     string `json:"mainName"`
	Environment  string `json:"environment"`
	LoginTime    string `json:"loginTime"`
	ElapsedTime  string `json:"elapsedTime"`
	TotalInstr   int    `json:"totalInstr"`
	InstrPerSec  int    `json:"instrPerSec"`
	Remark       string `json:"remark"`
	MemUsed      int    `json:"memUsed"`
	SID          string `json:"sid"`
	RPCState     int    `json:"rpcState"`
	AppUser      string `json:"appUser"`
}

// KillUserParams are the parameters for $totvsmonitor/killUser.
type KillUserParams struct {
	KillUserInfo struct {
		ConnectionToken string `json:"connectionToken"`
		UserName        string `json:"userName"`
		ComputerName    string `json:"computerName"`
		ThreadID        int    `json:"threadId"`
		ServerName      string `json:"serverName"`
	} `json:"killUserInfo"`
}

// SetConnectionStatusParams are the parameters for $totvsmonitor/setConnectionStatus.
type SetConnectionStatusParams struct {
	SetConnectionStatusInfo struct {
		ConnectionToken string `json:"connectionToken"`
		Status          bool   `json:"status"`
	} `json:"setConnectionStatusInfo"`
}

// ─── Authorization ──────────────────────────────────────────────────────────

// ValidKeyParams are the parameters for $totvsserver/validKey.
type ValidKeyParams struct {
	ValidKeyInfo struct {
		ConnectionToken string `json:"connectionToken"`
		MachineID       string `json:"machineId,omitempty"`
		TokenKey        string `json:"tokenKey,omitempty"`
	} `json:"validKeyInfo"`
}
