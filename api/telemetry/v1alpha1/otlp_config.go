package v1alpha1

import (
	"time"

	"go.opentelemetry.io/collector/config/configcompression"
)

type TimeoutSettings struct {
	// Timeout is the timeout for every attempt to send data to the backend.
	// A zero timeout means no timeout.
	Timeout time.Duration `json:"timeout,omitempty"`
}

// QueueSettings defines configuration for queueing batches before sending to the consumerSender.
type QueueSettings struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `json:"enabled,omitempty"`
	// NumConsumers is the number of consumers from the queue.
	NumConsumers int `json:"num_consumers,omitempty"`
	// QueueSize is the maximum number of batches allowed in queue at a given time.
	QueueSize int `json:"queue_size,omitempty"`
	// StorageID if not empty, enables the persistent storage and uses the component specified
	// as a storage extension for the persistent queue
	StorageID string `json:"storage,omitempty"` //TODO this is *component.ID at Otel
}

// BackOffConfig defines configuration for retrying batches in case of export failure.
// The current supported strategy is exponential backoff.
type BackOffConfig struct {
	// Enabled indicates whether to not retry sending batches in case of export failure.
	Enabled bool `json:"enabled,omitempty"`
	// InitialInterval the time to wait after the first failure before retrying.
	InitialInterval time.Duration `json:"initial_interval,omitempty"`
	// RandomizationFactor is a random factor used to calculate next backoffs
	// Randomized interval = RetryInterval * (1 ± RandomizationFactor)
	RandomizationFactor string `json:"randomization_factor,omitempty"`
	// Multiplier is the value multiplied by the backoff interval bounds
	Multiplier string `json:"multiplier,omitempty"`
	// MaxInterval is the upper bound on backoff interval. Once this value is reached the delay between
	// consecutive retries will always be `MaxInterval`.
	MaxInterval time.Duration `json:"max_interval,omitempty"`
	// MaxElapsedTime is the maximum amount of time (including retries) spent trying to send a request/batch.
	// Once this value is reached, the data is discarded. If set to 0, the retries are never stopped.
	MaxElapsedTime time.Duration `json:"max_elapsed_time,omitempty"`
}

// KeepaliveClientConfig exposes the keepalive.ClientParameters to be used by the exporter.
// Refer to the original data-structure for the meaning of each parameter:
// https://godoc.org/google.golang.org/grpc/keepalive#ClientParameters
type KeepaliveClientConfig struct {
	Time                time.Duration `json:"time,omitempty"`
	Timeout             time.Duration `json:"timeout,omitempty"`
	PermitWithoutStream bool          `json:"permit_without_stream,omitempty"`
}

// GRPCClientSettings defines common settings for a gRPC client configuration.
type GRPCClientSettings struct {
	// The target to which the exporter is going to send traces or metrics,
	// using the gRPC protocol. The valid syntax is described at
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	Endpoint string `json:"endpoint"`

	// The compression key for supported compression types within collector.
	Compression configcompression.CompressionType `json:"compression,omitempty"`

	// TLSSetting struct exposes TLS client configuration.
	TLSSetting TLSClientSetting `json:"tls,omitempty"`

	// The keepalive parameters for gRPC client. See grpc.WithKeepaliveParams.
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	Keepalive *KeepaliveClientConfig `json:"keepalive,omitempty"`

	// ReadBufferSize for gRPC client. See grpc.WithReadBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WithReadBufferSize).
	ReadBufferSize int `json:"read_buffer_size,omitempty"`

	// WriteBufferSize for gRPC gRPC. See grpc.WithWriteBufferSize.
	// (https://godoc.org/google.golang.org/grpc#WithWriteBufferSize).
	WriteBufferSize int `json:"write_buffer_size,omitempty"`

	// WaitForReady parameter configures client to wait for ready state before sending data.
	// (https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
	WaitForReady bool `json:"wait_for_ready,omitempty"`

	// The headers associated with gRPC requests.
	Headers map[string]string `json:"headers,omitempty"`

	// Sets the balancer in grpclb_policy to discover the servers. Default is pick_first.
	// https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md
	BalancerName string `json:"balancer_name,omitempty"`

	// WithAuthority parameter configures client to rewrite ":authority" header
	// (godoc.org/google.golang.org/grpc#WithAuthority)
	Authority string `json:"authority,omitempty"`

	// Auth configuration for outgoing RPCs.
	Auth string `json:"auth,omitempty"` //TODO this is a reference *configauth.Authentication
}

// TLSClientSetting contains TLS configurations that are specific to client
// connections in addition to the common configurations. This should be used by
// components configuring TLS client connections.
type TLSClientSetting struct {
	// squash ensures fields are correctly decoded in embedded struct.
	//TLSSetting `json:",inline"`

	// These are config options specific to client connections.

	// In gRPC when set to true, this is used to disable the client transport security.
	// See https://godoc.org/google.golang.org/grpc#WithInsecure.
	// In HTTP, this disables verifying the server's certificate chain and host name
	// (InsecureSkipVerify in the tls Config). Please refer to
	// https://godoc.org/crypto/tls#Config for more information.
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

	// ReloadInterval specifies the duration after which the certificate will be reloaded
	// If not set, it will never be reloaded (optional)
	ReloadInterval time.Duration `json:"reload_interval,omitempty"`
}
