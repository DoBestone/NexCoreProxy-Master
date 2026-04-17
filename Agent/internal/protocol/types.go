// Package protocol 与 Master 的 HTTP 协议数据结构（必须与 Master agent_protocol.go 对齐）
package protocol

// ServerConfig GET /api/v1/server/config 的响应（与 Master AgentConfig 对齐）
type ServerConfig struct {
	Etag     string         `json:"etag"`
	Node     NodeMeta       `json:"node"`
	Inbounds []Inbound      `json:"inbounds"`
	Users    []User         `json:"users"`
	Relays   []Relay        `json:"relays"`
	Settings RuntimeSetting `json:"settings"`
}

type NodeMeta struct {
	ID               uint   `json:"id"`
	Role             string `json:"role"`
	Region           string `json:"region"`
	AgentTarget      string `json:"agentTarget"`
	XrayTarget       string `json:"xrayTarget"`
	AgentDownloadURL string `json:"agentDownloadUrl,omitempty"`
	AgentSHA256URL   string `json:"agentSha256Url,omitempty"`
}

type RuntimeSetting struct {
	PullInterval int    `json:"pullInterval"`
	PushInterval int    `json:"pushInterval"`
	LogLevel     string `json:"logLevel"`
}

type Inbound struct {
	ID                uint   `json:"id"`
	Tag               string `json:"tag"`
	Protocol          string `json:"protocol"`
	Listen            string `json:"listen"`
	Port              int    `json:"port"`
	PortRange         string `json:"portRange,omitempty"`
	Network           string `json:"network"`
	Security          string `json:"security"`
	StreamJSON        string `json:"streamJson,omitempty"`
	TLSJSON           string `json:"tlsJson,omitempty"`
	SniffJSON         string `json:"sniffJson,omitempty"`
	SettingsJSON      string `json:"settingsJson,omitempty"`
	RealityPrivateKey string `json:"realityPrivateKey,omitempty"`
	RealityShortID    string `json:"realityShortId,omitempty"`
	RealitySNI        string `json:"realitySni,omitempty"`
	RealityDest       string `json:"realityDest,omitempty"`
	CertDomain        string `json:"certDomain,omitempty"`
	CertPEM           string `json:"certPem,omitempty"`
	KeyPEM            string `json:"keyPem,omitempty"`
}

type User struct {
	ID             uint   `json:"id"`
	Email          string `json:"email"`
	UUID           string `json:"uuid"`
	TrojanPassword string `json:"trojanPwd,omitempty"`
	SS2022Password string `json:"ssPwd,omitempty"`
	LimitIP        int    `json:"limitIp,omitempty"`
	SpeedMbps      int    `json:"speedMbps,omitempty"`
	ExpiryMs       int64  `json:"expiryMs,omitempty"`
	InboundIDs     []uint `json:"inboundIds"`
}

type Relay struct {
	ID         uint   `json:"id"`
	Tag        string `json:"tag"`
	Mode       string `json:"mode"`
	ListenPort int    `json:"listenPort"`
	PortRange  string `json:"portRange,omitempty"`

	BackendIP   string `json:"backendIp"`
	BackendPort int    `json:"backendPort"`

	WrapProtocol     string `json:"wrapProtocol,omitempty"`
	WrapStreamJSON   string `json:"wrapStreamJson,omitempty"`
	WrapTLSJSON      string `json:"wrapTlsJson,omitempty"`
	WrapRealityPriv  string `json:"wrapRealityPriv,omitempty"`
	WrapRealityShort string `json:"wrapRealityShort,omitempty"`
	TrunkUUID        string `json:"trunkUuid,omitempty"`
	TrunkPassword    string `json:"trunkPassword,omitempty"`

	BackendInboundProtocol string `json:"backendInboundProtocol,omitempty"`
	BackendInboundStream   string `json:"backendInboundStream,omitempty"`
	BackendInboundTLS      string `json:"backendInboundTls,omitempty"`
}

// PushRequest POST /api/v1/server/push 的请求
type PushRequest struct {
	EtagApplied string                  `json:"etagApplied"`
	Stats       map[string]TrafficDelta `json:"stats"`
	Online      map[string][]string     `json:"online"`
	System      *SystemSnapshot         `json:"system"`
}

type TrafficDelta struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

type SystemSnapshot struct {
	CPU          float64 `json:"cpu"`
	Mem          float64 `json:"mem"`
	Load         float64 `json:"load"`
	XrayUptime   uint64  `json:"xrayUptime"`
	XrayVersion  string  `json:"xrayVersion"`
	AgentVersion string  `json:"agentVersion"`
}

// PushResponse Master 的响应
type PushResponse struct {
	Success     bool     `json:"success"`
	Kicks       []string `json:"kicks"`
	CurrentEtag string   `json:"currentEtag"`
	Msg         string   `json:"msg,omitempty"`
}
