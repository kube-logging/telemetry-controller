// Copyright © 2023 Kube logging authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"time"

	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/config/configopaque"
)

// TimeoutSettings for timeout. The timeout applies to individual attempts to send data to the backend.
type TimeoutSettings struct {
	// Timeout is the timeout for every attempt to send data to the backend.
	// A zero timeout means no timeout.
	Timeout *time.Duration `json:"timeout,omitempty"`
}

// QueueSettings defines configuration for queueing batches before sending to the consumerSender.
type QueueSettings struct {
	// NumConsumers is the number of consumers from the queue. Defaults to 10.
	// If batching is enabled, a combined batch cannot contain more requests than the number of consumers.
	// So it's recommended to set higher number of consumers if batching is enabled.
	NumConsumers *int `json:"num_consumers,omitempty"`

	// QueueSize is the maximum number of batches allowed in queue at a given time.
	// Default value is 100.
	QueueSize *int `json:"queue_size,omitempty"`

	// Blocking controls the queue behavior when full.
	// If true it blocks until enough space to add the new request to the queue.
	Blocking *bool `json:"blocking,omitempty"`
}

// BackOffConfig defines configuration for retrying batches in case of export failure.
// The current supported strategy is exponential backoff.
type BackOffConfig struct {
	// InitialInterval the time to wait after the first failure before retrying.
	InitialInterval *time.Duration `json:"initial_interval,omitempty"`

	// RandomizationFactor is a random factor used to calculate next backoffs
	// Randomized interval = RetryInterval * (1 ± RandomizationFactor)
	RandomizationFactor *string `json:"randomization_factor,omitempty"`

	// Multiplier is the value multiplied by the backoff interval bounds
	Multiplier *string `json:"multiplier,omitempty"`

	// MaxInterval is the upper bound on backoff interval. Once this value is reached the delay between
	// consecutive retries will always be `MaxInterval`.
	MaxInterval *time.Duration `json:"max_interval,omitempty"`

	// MaxElapsedTime is the maximum amount of time (including retries) spent trying to send a request/batch.
	// Once this value is reached, the data is discarded. If set to 0, the retries are never stopped.
	// Default value is 0 to ensure that the data is never discarded.
	MaxElapsedTime *time.Duration `json:"max_elapsed_time,omitempty"`
}

// KeepaliveClientConfig exposes the keepalive.ClientParameters to be used by the exporter.
// Refer to the original data-structure for the meaning of each parameter:
// https://godoc.org/google.golang.org/grpc/keepalive#ClientParameters
type KeepaliveClientConfig struct {
	Time                time.Duration `json:"time,omitempty"`
	Timeout             time.Duration `json:"timeout,omitempty"`
	PermitWithoutStream bool          `json:"permit_without_stream,omitempty"`
}

// ClientConfig defines common settings for a gRPC client configuration.
type GRPCClientConfig struct {
	// The target to which the exporter is going to send traces or metrics,
	// using the gRPC protocol. The valid syntax is described at
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	Endpoint *string `json:"endpoint"`

	// The compression key for supported compression types within collector.
	Compression *configcompression.Type `json:"compression,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *TLSClientSetting `json:"tls,omitempty"`

	// The keepalive parameters for gRPC client. See grpc.WithKeepaliveParams.
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	Keepalive *KeepaliveClientConfig `json:"keepalive,omitempty"`

	// ReadBufferSize for gRPC client. See grpc.WithReadBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WithReadBufferSize).
	ReadBufferSize *int `json:"read_buffer_size,omitempty"`

	// WriteBufferSize for gRPC gRPC. See grpc.WithWriteBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WithWriteBufferSize).
	WriteBufferSize *int `json:"write_buffer_size,omitempty"`

	// WaitForReady parameter configures client to wait for ready state before sending data.
	// (https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
	WaitForReady *bool `json:"wait_for_ready,omitempty"`

	// The headers associated with gRPC requests.
	Headers *map[string]string `json:"headers,omitempty"`

	// Sets the balancer in grpclb_policy to discover the servers. Default is pick_first.
	// https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md
	BalancerName *string `json:"balancer_name,omitempty"`

	// WithAuthority parameter configures client to rewrite ":authority" header
	// (godoc.org/google.golang.org/grpc#WithAuthority)
	Authority *string `json:"authority,omitempty"`

	// Auth configuration for outgoing RPCs.
	Auth *Authentication `json:"auth,omitempty"`
}

// TLSClientSetting contains TLS configurations that are specific to client
// connections in addition to the common configurations. This should be used by
// components configuring TLS client connections.
type TLSClientSetting struct {
	// squash ensures fields are correctly decoded in embedded struct.
	TLSSetting `json:",inline"`

	// These are config options specific to client connections.

	// In gRPC and HTTP when set to true, this is used to disable the client transport security.
	// See https://godoc.org/google.golang.org/grpc#WithInsecure for gRPC.
	// Please refer to https://godoc.org/crypto/tls#Config for more information.
	// (optional, default false)
	Insecure bool `json:"insecure,omitempty"`

	// InsecureSkipVerify will enable TLS but not verify the certificate.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`

	// ServerName requested by client for virtual hosting.
	// This sets the ServerName in the TLSConfig. Please refer to
	// https://godoc.org/crypto/tls#Config for more information. (optional)
	ServerName string `json:"server_name_override,omitempty"`
}

// TLSSetting exposes the common client and server TLS configurations.
// Note: Since there isn't anything specific to a server connection. Components
// with server connections should use TLSSetting.
type TLSSetting struct {
	// Path to the CA cert. For a client this verifies the server certificate.
	// For a server this verifies client certificates. If empty uses system root CA.
	// (optional)
	CAFile string `json:"ca_file,omitempty"`

	// In memory PEM encoded cert. (optional)
	CAPem string `json:"ca_pem,omitempty"`

	// If true, load system CA certificates pool in addition to the certificates
	// configured in this struct.
	IncludeSystemCACertsPool bool `json:"include_system_ca_certs_pool,omitempty"`

	// Path to the TLS cert to use for TLS required connections. (optional)
	CertFile string `json:"cert_file,omitempty"`

	// In memory PEM encoded TLS cert to use for TLS required connections. (optional)
	CertPem string `json:"cert_pem,omitempty"`

	// Path to the TLS key to use for TLS required connections. (optional)
	KeyFile string `json:"key_file,omitempty"`

	// In memory PEM encoded TLS key to use for TLS required connections. (optional)
	KeyPem string `json:"key_pem,omitempty"`

	// MinVersion sets the minimum TLS version that is acceptable.
	// If not set, TLS 1.2 will be used. (optional)
	MinVersion string `json:"min_version,omitempty"`

	// MaxVersion sets the maximum TLS version that is acceptable.
	// If not set, refer to crypto/tls for defaults. (optional)
	MaxVersion string `json:"max_version,omitempty"`

	// CipherSuites is a list of TLS cipher suites that the TLS transport can use.
	// If left blank, a safe default list is used.
	// See https://go.dev/src/crypto/tls/cipher_suites.go for a list of supported cipher suites.
	CipherSuites []string `json:"cipher_suites,omitempty"`

	// ReloadInterval specifies the duration after which the certificate will be reloaded
	// If not set, it will never be reloaded (optional)
	ReloadInterval time.Duration `json:"reload_interval,omitempty"`

	// contains the elliptic curves that will be used in
	// an ECDHE handshake, in preference order
	// Defaults to empty list and "crypto/tls" defaults are used, internally.
	CurvePreferences []string `json:"curve_preferences,omitempty"`
}

type Authentication struct {
	// AuthenticatorID specifies the name of the extension to use in order to authenticate the incoming data point.
	AuthenticatorID *string `json:"authenticator,omitempty"`
}

type CompressionParams struct {
	Level *int `json:"level,omitempty"`
}

// CookiesConfig defines the configuration of the HTTP client regarding cookies served by the server.
type CookiesConfig struct {
	// Enabled if true, cookies from HTTP responses will be reused in further HTTP requests with the same server.
	Enabled bool `json:"enabled,omitempty"`
}

// HTTPClientConfig defines settings for creating an HTTP client.
type HTTPClientConfig struct {
	// The target URL to send data to (e.g.: http://some.url:9411/v1/traces).
	Endpoint *string `json:"endpoint,omitempty"`

	// ProxyURL setting for the collector
	ProxyURL *string `json:"proxy_url,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting *TLSClientSetting `json:"tls,omitempty"`

	// ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.
	// Default is 0.
	ReadBufferSize *int `json:"read_buffer_size,omitempty"`

	// WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.
	// Default is 0.
	WriteBufferSize *int `json:"write_buffer_size,omitempty"`

	// Timeout parameter configures `http.Client.Timeout`.
	// Default is 0 (unlimited).
	Timeout *time.Duration `json:"timeout,omitempty"`

	// Additional headers attached to each HTTP request sent by the client.
	// Existing header values are overwritten if collision happens.
	// Header values are opaque since they may be sensitive.
	Headers *map[string]configopaque.String `json:"headers,omitempty"`

	// Auth configuration for outgoing HTTP calls.
	Auth *Authentication `json:"auth,omitempty"`

	// The compression key for supported compression types within collector.
	Compression *configcompression.Type `json:"compression,omitempty"`

	// Advanced configuration options for the Compression
	CompressionParams *CompressionParams `json:"compression_params,omitempty"`

	// MaxIdleConns is used to set a limit to the maximum idle HTTP connections the client can keep open.
	// By default, it is set to 100.
	MaxIdleConns *int `json:"max_idle_conns,omitempty"`

	// MaxIdleConnsPerHost is used to set a limit to the maximum idle HTTP connections the host can keep open.
	// By default, it is set to [http.DefaultTransport.MaxIdleConnsPerHost].
	MaxIdleConnsPerHost *int `json:"max_idle_conns_per_host,omitempty"`

	// MaxConnsPerHost limits the total number of connections per host, including connections in the dialing,
	// active, and idle states.
	// By default, it is set to [http.DefaultTransport.MaxConnsPerHost].
	MaxConnsPerHost *int `json:"max_conns_per_host,omitempty"`

	// IdleConnTimeout is the maximum amount of time a connection will remain open before closing itself.
	// By default, it is set to [http.DefaultTransport.IdleConnTimeout]
	IdleConnTimeout *time.Duration `json:"idle_conn_timeout,omitempty"`

	// DisableKeepAlives, if true, disables HTTP keep-alives and will only use the connection to the server
	// for a single HTTP request.
	//
	// WARNING: enabling this option can result in significant overhead establishing a new HTTP(S)
	// connection for every request. Before enabling this option please consider whether changes
	// to idle connection settings can achieve your goal.
	DisableKeepAlives *bool `json:"disable_keep_alives,omitempty"`

	// This is needed in case you run into
	// https://github.com/golang/go/issues/59690
	// https://github.com/golang/go/issues/36026
	// HTTP2ReadIdleTimeout if the connection has been idle for the configured value send a ping frame for health check
	// 0s means no health check will be performed.
	HTTP2ReadIdleTimeout *time.Duration `json:"http2_read_idle_timeout,omitempty"`

	// HTTP2PingTimeout if there's no response to the ping within the configured value, the connection will be closed.
	// If not set or set to 0, it defaults to 15s.
	HTTP2PingTimeout *time.Duration `json:"http2_ping_timeout,omitempty"`

	// Cookies configures the cookie management of the HTTP client.
	Cookies *CookiesConfig `json:"cookies,omitempty"`
}
