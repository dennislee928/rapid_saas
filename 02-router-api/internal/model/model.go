package model

import "encoding/json"

type Endpoint struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	Name          string `json:"name"`
	SourcePreset  string `json:"source_preset,omitempty"`
	SigningSecret string `json:"signing_secret,omitempty"`
	SigningHeader string `json:"signing_header,omitempty"`
	SigningAlgo   string `json:"signing_algo,omitempty"`
	Enabled       bool   `json:"enabled"`
	CreatedAt     int64  `json:"created_at"`
}

type Destination struct {
	ID         string          `json:"id"`
	TenantID   string          `json:"tenant_id"`
	Kind       string          `json:"kind"`
	Name       string          `json:"name"`
	ConfigJSON json.RawMessage `json:"config_json"`
	SecretRef  string          `json:"secret_ref,omitempty"`
	CreatedAt  int64           `json:"created_at"`
}

type DestinationConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

type Rule struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	EndpointID      string          `json:"endpoint_id"`
	Position        int             `json:"position"`
	Name            string          `json:"name"`
	FilterJSONLogic json.RawMessage `json:"filter_jsonlogic,omitempty"`
	TransformKind   string          `json:"transform_kind"`
	TransformBody   string          `json:"transform_body,omitempty"`
	DestinationID   string          `json:"destination_id,omitempty"`
	OnMatch         string          `json:"on_match"`
	Enabled         bool            `json:"enabled"`
	CreatedAt       int64           `json:"created_at"`
}

type QueueEvent struct {
	ID          string            `json:"id"`
	TenantID    string            `json:"tenant_id"`
	EndpointID  string            `json:"endpoint_id"`
	Headers     map[string]string `json:"headers"`
	Body        json.RawMessage   `json:"body"`
	ReceivedAt  int64             `json:"received_at_ms"`
	RequestHash string            `json:"request_hash"`
}

type DeliveryLog struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	EndpointID    string `json:"endpoint_id"`
	RuleID        string `json:"rule_id,omitempty"`
	DestinationID string `json:"destination_id,omitempty"`
	Status        string `json:"status"`
	Attempt       int    `json:"attempt"`
	HTTPStatus    int    `json:"http_status,omitempty"`
	LatencyMS     int64  `json:"latency_ms,omitempty"`
	Error         string `json:"error,omitempty"`
	RequestHash   string `json:"request_hash,omitempty"`
	RequestSize   int    `json:"request_size,omitempty"`
	ReceivedAt    int64  `json:"received_at"`
	DeliveredAt   int64  `json:"delivered_at,omitempty"`
}

type DLQEntry struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	EndpointID    string `json:"endpoint_id"`
	RuleID        string `json:"rule_id,omitempty"`
	DestinationID string `json:"destination_id,omitempty"`
	PayloadB64    string `json:"payload_b64"`
	LastError     string `json:"last_error,omitempty"`
	Attempts      int    `json:"attempts"`
	ParkedAt      int64  `json:"parked_at"`
}

type UsageSummary struct {
	TenantID  string `json:"tenant_id"`
	Window    string `json:"window"`
	Ingressed int64  `json:"ingressed"`
	Forwarded int64  `json:"forwarded"`
	Failed    int64  `json:"failed"`
}
