---
title: API Reference
---

## Packages
- [telemetry.kube-logging.dev/v1alpha1](#telemetrykube-loggingdevv1alpha1)


## telemetry.kube-logging.dev/v1alpha1

Package v1alpha1 contains API Schema definitions for the telemetry v1alpha1 API group

- [Bridge](#bridge)
- [BridgeList](#bridgelist)
- [Collector](#collector)
- [CollectorList](#collectorlist)
- [Output](#output)
- [OutputList](#outputlist)
- [Subscription](#subscription)
- [SubscriptionList](#subscriptionlist)
- [Tenant](#tenant)
- [TenantList](#tenantlist)



### Authentication





_Appears in:_
- [GRPCClientConfig](#grpcclientconfig)
- [HTTPClientConfig](#httpclientconfig)
- [OTLPGRPC](#otlpgrpc)
- [OTLPHTTP](#otlphttp)

#### `authenticator` (_string_)

AuthenticatorID specifies the name of the extension to use in order to authenticate the incoming data point.


### BackOffConfig



BackOffConfig defines configuration for retrying batches in case of export failure.
The current supported strategy is exponential backoff.

_Appears in:_
- [Fluentforward](#fluentforward)
- [OTLPGRPC](#otlpgrpc)
- [OTLPHTTP](#otlphttp)

#### `enabled` (_boolean_)

Enabled indicates whether to not retry sending batches in case of export failure.
#### `initial_interval` (_[Duration](#duration)_)

InitialInterval the time to wait after the first failure before retrying.
#### `randomization_factor` (_string_)

RandomizationFactor is a random factor used to calculate next backoffs
Randomized interval = RetryInterval * (1 ± RandomizationFactor)
#### `multiplier` (_string_)

Multiplier is the value multiplied by the backoff interval bounds
#### `max_interval` (_[Duration](#duration)_)

MaxInterval is the upper bound on backoff interval. Once this value is reached the delay between
consecutive retries will always be `MaxInterval`.
#### `max_elapsed_time` (_[Duration](#duration)_)

MaxElapsedTime is the maximum amount of time (including retries) spent trying to send a request/batch.
Once this value is reached, the data is discarded. If set to 0, the retries are never stopped.


### BasicAuthConfig





_Appears in:_
- [OutputAuth](#outputauth)

#### `secretRef` (_[SecretReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#secretreference-v1-core)_)


#### `usernameField` (_string_)


#### `passwordField` (_string_)




### BearerAuthConfig





_Appears in:_
- [OutputAuth](#outputauth)

#### `secretRef` (_[SecretReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#secretreference-v1-core)_)


#### `tokenField` (_string_)




### Bridge



Bridge is the Schema for the Bridges API

_Appears in:_
- [BridgeList](#bridgelist)

<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `Bridge` </b>
#### `metadata` (_[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `spec` (_[BridgeSpec](#bridgespec)_)




### BridgeList



BridgeList contains a list of Bridge



<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `BridgeList` </b>
#### `metadata` (_[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#listmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `items` (_[Bridge](#bridge) array_)




### BridgeSpec



BridgeSpec defines the desired state of Bridge

_Appears in:_
- [Bridge](#bridge)

#### `sourceTenant` (_string_)


#### `targetTenant` (_string_)


#### `condition` (_string_)

The OTTL condition which must be satisfied in order to forward telemetry
from the source tenant to the target tenant




### Collector



Collector is the Schema for the collectors API

_Appears in:_
- [CollectorList](#collectorlist)

<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `Collector` </b>
#### `metadata` (_[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `spec` (_[CollectorSpec](#collectorspec)_)




### CollectorList



CollectorList contains a list of Collector



<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `CollectorList` </b>
#### `metadata` (_[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#listmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `items` (_[Collector](#collector) array_)




### CollectorSpec



CollectorSpec defines the desired state of Collector

_Appears in:_
- [Collector](#collector)

#### `tenantSelector` (_[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta)_)


#### `controlNamespace` (_string_)

Namespace where OTel collector DaemonSet is deployed
#### `debug` (_boolean_)

Enables debug logging for the collector
#### `memoryLimiter` (_[MemoryLimiter](#memorylimiter)_)

Setting memory limits for the Collector
#### `daemonSet` (_[DaemonSet](#daemonset)_)






### Fluentforward





_Appears in:_
- [OutputSpec](#outputspec)

#### `endpoint` (_string_)

The target endpoint URI to send data to (e.g.: some.url:24224).
#### `connection_timeout` (_[Duration](#duration)_)

Connection Timeout parameter configures `net.Dialer`.
#### `tls` (_[TLSClientSetting](#tlsclientsetting)_)

TLSSetting struct exposes TLS client configuration.
#### `shared_key` (_string_)

SharedKey is used for authorization with the server that knows it.
#### `require_ack` (_boolean_)

RequireAck enables the acknowledgement feature.
#### `tag` (_string_)

The Fluent tag parameter used for routing
#### `compress_gzip` (_boolean_)

CompressGzip enables gzip compression for the payload.
#### `default_labels_enabled` (_object (keys:string, values:boolean)_)

DefaultLabelsEnabled is a map of default attributes to be added to each log record.
#### `sending_queue` (_[QueueSettings](#queuesettings)_)


#### `retry_on_failure` (_[BackOffConfig](#backoffconfig)_)


#### `kubernetes_metadata` (_[KubernetesMetadata](#kubernetesmetadata)_)




### GRPCClientConfig



ClientConfig defines common settings for a gRPC client configuration.

_Appears in:_
- [OTLPGRPC](#otlpgrpc)

#### `endpoint` (_string_)

The target to which the exporter is going to send traces or metrics,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md.
#### `compression` (_[Type](#type)_)

The compression key for supported compression types within collector.
#### `tls` (_[TLSClientSetting](#tlsclientsetting)_)

TLSSetting struct exposes TLS client configuration.
#### `keepalive` (_[KeepaliveClientConfig](#keepaliveclientconfig)_)

The keepalive parameters for gRPC client. See grpc.WithKeepaliveParams.
(https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
#### `read_buffer_size` (_integer_)

ReadBufferSize for gRPC client. See grpc.WithReadBufferSize.
(https://godoc.org/google.golang.org/grpc#WithReadBufferSize).
#### `write_buffer_size` (_integer_)

WriteBufferSize for gRPC gRPC. See grpc.WithWriteBufferSize.
(https://godoc.org/google.golang.org/grpc#WithWriteBufferSize).
#### `wait_for_ready` (_boolean_)

WaitForReady parameter configures client to wait for ready state before sending data.
(https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
#### `headers` (_object (keys:string, values:string)_)

The headers associated with gRPC requests.
#### `balancer_name` (_string_)

Sets the balancer in grpclb_policy to discover the servers. Default is pick_first.
https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md
#### `authority` (_string_)

WithAuthority parameter configures client to rewrite ":authority" header
(godoc.org/google.golang.org/grpc#WithAuthority)
#### `auth` (_[Authentication](#authentication)_)

Auth configuration for outgoing RPCs.


### HTTPClientConfig



ClientConfig defines settings for creating an HTTP client.

_Appears in:_
- [OTLPHTTP](#otlphttp)

#### `endpoint` (_string_)

The target URL to send data to (e.g.: http://some.url:9411/v1/traces).
#### `proxy_url` (_string_)

ProxyURL setting for the collector
#### `tls` (_[TLSClientSetting](#tlsclientsetting)_)

TLSSetting struct exposes TLS client configuration.
#### `read_buffer_size` (_integer_)

ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.
#### `write_buffer_size` (_integer_)

WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.
#### `timeout` (_[Duration](#duration)_)

Timeout parameter configures `http.Client.Timeout`.
#### `headers` (_object (keys:string, values:String)_)

Additional headers attached to each HTTP request sent by the client.
Existing header values are overwritten if collision happens.
Header values are opaque since they may be sensitive.
#### `auth` (_[Authentication](#authentication)_)

Auth configuration for outgoing HTTP calls.
#### `compression` (_[Type](#type)_)

The compression key for supported compression types within collector.
#### `max_idle_conns` (_integer_)

MaxIdleConns is used to set a limit to the maximum idle HTTP connections the client can keep open.
There's an already set value, and we want to override it only if an explicit value provided
#### `max_idle_conns_per_host` (_integer_)

MaxIdleConnsPerHost is used to set a limit to the maximum idle HTTP connections the host can keep open.
There's an already set value, and we want to override it only if an explicit value provided
#### `max_conns_per_host` (_integer_)

MaxConnsPerHost limits the total number of connections per host, including connections in the dialing,
active, and idle states.
There's an already set value, and we want to override it only if an explicit value provided
#### `idle_conn_timeout` (_[Duration](#duration)_)

IdleConnTimeout is the maximum amount of time a connection will remain open before closing itself.
There's an already set value, and we want to override it only if an explicit value provided
#### `disable_keep_alives` (_boolean_)

DisableKeepAlives, if true, disables HTTP keep-alives and will only use the connection to the server
for a single HTTP request.


WARNING: enabling this option can result in significant overhead establishing a new HTTP(S)
connection for every request. Before enabling this option please consider whether changes
to idle connection settings can achieve your goal.
#### `http2_read_idle_timeout` (_[Duration](#duration)_)

This is needed in case you run into
https://github.com/golang/go/issues/59690
https://github.com/golang/go/issues/36026
HTTP2ReadIdleTimeout if the connection has been idle for the configured value send a ping frame for health check
0s means no health check will be performed.
#### `http2_ping_timeout` (_[Duration](#duration)_)

HTTP2PingTimeout if there's no response to the ping within the configured value, the connection will be closed.
If not set or set to 0, it defaults to 15s.


### KeepaliveClientConfig



KeepaliveClientConfig exposes the keepalive.ClientParameters to be used by the exporter.
Refer to the original data-structure for the meaning of each parameter:
https://godoc.org/google.golang.org/grpc/keepalive#ClientParameters

_Appears in:_
- [GRPCClientConfig](#grpcclientconfig)
- [OTLPGRPC](#otlpgrpc)

#### `time` (_[Duration](#duration)_)


#### `timeout` (_[Duration](#duration)_)


#### `permit_without_stream` (_boolean_)




### KubernetesMetadata





_Appears in:_
- [Fluentforward](#fluentforward)

#### `key` (_string_)


#### `include_pod_labels` (_boolean_)




### MemoryLimiter





_Appears in:_
- [CollectorSpec](#collectorspec)

#### `check_interval` (_[Duration](#duration)_)

CheckInterval is the time between measurements of memory usage for the
purposes of avoiding going over the limits. Defaults to zero, so no
checks will be performed.
#### `limit_mib` (_integer_)

MemoryLimitMiB is the maximum amount of memory, in MiB, targeted to be
allocated by the process.
#### `spike_limit_mib` (_integer_)

MemorySpikeLimitMiB is the maximum, in MiB, spike expected between the
measurements of memory usage.
#### `limit_percentage` (_integer_)

MemoryLimitPercentage is the maximum amount of memory, in %, targeted to be
allocated by the process. The fixed memory settings MemoryLimitMiB has a higher precedence.
#### `spike_limit_percentage` (_integer_)

MemorySpikePercentage is the maximum, in percents against the total memory,
spike expected between the measurements of memory usage.


### NamespacedName





_Appears in:_
- [SubscriptionSpec](#subscriptionspec)
- [SubscriptionStatus](#subscriptionstatus)
- [TenantStatus](#tenantstatus)

#### `namespace` (_string_)


#### `name` (_string_)




### OTLPGRPC



OTLP grpc exporter config ref: https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/otlpexporter/config.go

_Appears in:_
- [OutputSpec](#outputspec)

#### `sending_queue` (_[QueueSettings](#queuesettings)_)


#### `retry_on_failure` (_[BackOffConfig](#backoffconfig)_)


#### `timeout` (_[Duration](#duration)_)

Timeout is the timeout for every attempt to send data to the backend.
A zero timeout means no timeout.
#### `endpoint` (_string_)

The target to which the exporter is going to send traces or metrics,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md.
#### `compression` (_[Type](#type)_)

The compression key for supported compression types within collector.
#### `tls` (_[TLSClientSetting](#tlsclientsetting)_)

TLSSetting struct exposes TLS client configuration.
#### `keepalive` (_[KeepaliveClientConfig](#keepaliveclientconfig)_)

The keepalive parameters for gRPC client. See grpc.WithKeepaliveParams.
(https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
#### `read_buffer_size` (_integer_)

ReadBufferSize for gRPC client. See grpc.WithReadBufferSize.
(https://godoc.org/google.golang.org/grpc#WithReadBufferSize).
#### `write_buffer_size` (_integer_)

WriteBufferSize for gRPC gRPC. See grpc.WithWriteBufferSize.
(https://godoc.org/google.golang.org/grpc#WithWriteBufferSize).
#### `wait_for_ready` (_boolean_)

WaitForReady parameter configures client to wait for ready state before sending data.
(https://github.com/grpc/grpc/blob/master/doc/wait-for-ready.md)
#### `headers` (_object (keys:string, values:string)_)

The headers associated with gRPC requests.
#### `balancer_name` (_string_)

Sets the balancer in grpclb_policy to discover the servers. Default is pick_first.
https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md
#### `authority` (_string_)

WithAuthority parameter configures client to rewrite ":authority" header
(godoc.org/google.golang.org/grpc#WithAuthority)
#### `auth` (_[Authentication](#authentication)_)

Auth configuration for outgoing RPCs.


### OTLPHTTP





_Appears in:_
- [OutputSpec](#outputspec)

#### `sending_queue` (_[QueueSettings](#queuesettings)_)


#### `retry_on_failure` (_[BackOffConfig](#backoffconfig)_)


#### `endpoint` (_string_)

The target URL to send data to (e.g.: http://some.url:9411/v1/traces).
#### `proxy_url` (_string_)

ProxyURL setting for the collector
#### `tls` (_[TLSClientSetting](#tlsclientsetting)_)

TLSSetting struct exposes TLS client configuration.
#### `read_buffer_size` (_integer_)

ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.
#### `write_buffer_size` (_integer_)

WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.
#### `timeout` (_[Duration](#duration)_)

Timeout parameter configures `http.Client.Timeout`.
#### `headers` (_object (keys:string, values:String)_)

Additional headers attached to each HTTP request sent by the client.
Existing header values are overwritten if collision happens.
Header values are opaque since they may be sensitive.
#### `auth` (_[Authentication](#authentication)_)

Auth configuration for outgoing HTTP calls.
#### `compression` (_[Type](#type)_)

The compression key for supported compression types within collector.
#### `max_idle_conns` (_integer_)

MaxIdleConns is used to set a limit to the maximum idle HTTP connections the client can keep open.
There's an already set value, and we want to override it only if an explicit value provided
#### `max_idle_conns_per_host` (_integer_)

MaxIdleConnsPerHost is used to set a limit to the maximum idle HTTP connections the host can keep open.
There's an already set value, and we want to override it only if an explicit value provided
#### `max_conns_per_host` (_integer_)

MaxConnsPerHost limits the total number of connections per host, including connections in the dialing,
active, and idle states.
There's an already set value, and we want to override it only if an explicit value provided
#### `idle_conn_timeout` (_[Duration](#duration)_)

IdleConnTimeout is the maximum amount of time a connection will remain open before closing itself.
There's an already set value, and we want to override it only if an explicit value provided
#### `disable_keep_alives` (_boolean_)

DisableKeepAlives, if true, disables HTTP keep-alives and will only use the connection to the server
for a single HTTP request.


WARNING: enabling this option can result in significant overhead establishing a new HTTP(S)
connection for every request. Before enabling this option please consider whether changes
to idle connection settings can achieve your goal.
#### `http2_read_idle_timeout` (_[Duration](#duration)_)

This is needed in case you run into
https://github.com/golang/go/issues/59690
https://github.com/golang/go/issues/36026
HTTP2ReadIdleTimeout if the connection has been idle for the configured value send a ping frame for health check
0s means no health check will be performed.
#### `http2_ping_timeout` (_[Duration](#duration)_)

HTTP2PingTimeout if there's no response to the ping within the configured value, the connection will be closed.
If not set or set to 0, it defaults to 15s.


### Output



Output is the Schema for the outputs API

_Appears in:_
- [OutputList](#outputlist)

<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `Output` </b>
#### `metadata` (_[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `spec` (_[OutputSpec](#outputspec)_)




### OutputAuth





_Appears in:_
- [OutputSpec](#outputspec)

#### `basicauth` (_[BasicAuthConfig](#basicauthconfig)_)


#### `bearerauth` (_[BearerAuthConfig](#bearerauthconfig)_)




### OutputList



OutputList contains a list of Output



<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `OutputList` </b>
#### `metadata` (_[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#listmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `items` (_[Output](#output) array_)




### OutputSpec



OutputSpec defines the desired state of Output

_Appears in:_
- [Output](#output)

#### `otlp` (_[OTLPGRPC](#otlpgrpc)_)


#### `fluentforward` (_[Fluentforward](#fluentforward)_)


#### `otlphttp` (_[OTLPHTTP](#otlphttp)_)


#### `authentication` (_[OutputAuth](#outputauth)_)






### QueueSettings



QueueSettings defines configuration for queueing batches before sending to the consumerSender.

_Appears in:_
- [Fluentforward](#fluentforward)
- [OTLPGRPC](#otlpgrpc)
- [OTLPHTTP](#otlphttp)

#### `enabled` (_boolean_)

Enabled indicates whether to not enqueue batches before sending to the consumerSender.
#### `num_consumers` (_integer_)

NumConsumers is the number of consumers from the queue.
#### `queue_size` (_integer_)

QueueSize is the maximum number of batches allowed in queue at a given time.
#### `storage` (_string_)

StorageID if not empty, enables the persistent storage and uses the component specified
as a storage extension for the persistent queue


### RouteConfig



RouteConfig defines the routing configuration for a tenant
it will be used to generate routing connectors

_Appears in:_
- [TenantSpec](#tenantspec)

#### `defaultPipelines` (_string array_)


#### `errorMode` (_string_)

ErrorMode specifies how errors are handled while processing a statement
vaid options are: ignore, silent, propagate; (default: propagate)
#### `matchOnce` (_boolean_)




### State

_Underlying type:_ `string`



_Appears in:_
- [BridgeStatus](#bridgestatus)
- [CollectorStatus](#collectorstatus)
- [SubscriptionStatus](#subscriptionstatus)
- [TenantStatus](#tenantstatus)



### Subscription



Subscription is the Schema for the subscriptions API

_Appears in:_
- [SubscriptionList](#subscriptionlist)

<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `Subscription` </b>
#### `metadata` (_[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `spec` (_[SubscriptionSpec](#subscriptionspec)_)




### SubscriptionList



SubscriptionList contains a list of Subscription



<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `SubscriptionList` </b>
#### `metadata` (_[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#listmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `items` (_[Subscription](#subscription) array_)




### SubscriptionSpec



SubscriptionSpec defines the desired state of Subscription

_Appears in:_
- [Subscription](#subscription)

#### `condition` (_string_)


#### `outputs` (_[NamespacedName](#namespacedname) array_)






### TCPClientSettings





_Appears in:_
- [Fluentforward](#fluentforward)

#### `endpoint` (_string_)

The target endpoint URI to send data to (e.g.: some.url:24224).
#### `connection_timeout` (_[Duration](#duration)_)

Connection Timeout parameter configures `net.Dialer`.
#### `tls` (_[TLSClientSetting](#tlsclientsetting)_)

TLSSetting struct exposes TLS client configuration.
#### `shared_key` (_string_)

SharedKey is used for authorization with the server that knows it.


### TLSClientSetting



TLSClientSetting contains TLS configurations that are specific to client
connections in addition to the common configurations. This should be used by
components configuring TLS client connections.

_Appears in:_
- [Fluentforward](#fluentforward)
- [GRPCClientConfig](#grpcclientconfig)
- [HTTPClientConfig](#httpclientconfig)
- [OTLPGRPC](#otlpgrpc)
- [OTLPHTTP](#otlphttp)
- [TCPClientSettings](#tcpclientsettings)

#### `ca_file` (_string_)

Path to the CA cert. For a client this verifies the server certificate.
For a server this verifies client certificates. If empty uses system root CA.
(optional)
#### `ca_pem` (_string_)

In memory PEM encoded cert. (optional)
#### `cert_file` (_string_)

Path to the TLS cert to use for TLS required connections. (optional)
#### `cert_pem` (_string_)

In memory PEM encoded TLS cert to use for TLS required connections. (optional)
#### `key_file` (_string_)

Path to the TLS key to use for TLS required connections. (optional)
#### `key_pem` (_string_)

In memory PEM encoded TLS key to use for TLS required connections. (optional)
#### `min_version` (_string_)

MinVersion sets the minimum TLS version that is acceptable.
If not set, TLS 1.2 will be used. (optional)
#### `max_version` (_string_)

MaxVersion sets the maximum TLS version that is acceptable.
If not set, refer to crypto/tls for defaults. (optional)
#### `reload_interval` (_[Duration](#duration)_)

ReloadInterval specifies the duration after which the certificate will be reloaded
If not set, it will never be reloaded (optional)
#### `insecure` (_boolean_)

In gRPC when set to true, this is used to disable the client transport security.
See https://godoc.org/google.golang.org/grpc#WithInsecure.
In HTTP, this disables verifying the server's certificate chain and host name
(InsecureSkipVerify in the tls Config). Please refer to
https://godoc.org/crypto/tls#Config for more information.
(optional, default false)
#### `insecure_skip_verify` (_boolean_)

InsecureSkipVerify will enable TLS but not verify the certificate.
#### `server_name_override` (_string_)

ServerName requested by client for virtual hosting.
This sets the ServerName in the TLSConfig. Please refer to
https://godoc.org/crypto/tls#Config for more information. (optional)


### TLSSetting



TLSSetting exposes the common client and server TLS configurations.
Note: Since there isn't anything specific to a server connection. Components
with server connections should use TLSSetting.

_Appears in:_
- [TLSClientSetting](#tlsclientsetting)

#### `ca_file` (_string_)

Path to the CA cert. For a client this verifies the server certificate.
For a server this verifies client certificates. If empty uses system root CA.
(optional)
#### `ca_pem` (_string_)

In memory PEM encoded cert. (optional)
#### `cert_file` (_string_)

Path to the TLS cert to use for TLS required connections. (optional)
#### `cert_pem` (_string_)

In memory PEM encoded TLS cert to use for TLS required connections. (optional)
#### `key_file` (_string_)

Path to the TLS key to use for TLS required connections. (optional)
#### `key_pem` (_string_)

In memory PEM encoded TLS key to use for TLS required connections. (optional)
#### `min_version` (_string_)

MinVersion sets the minimum TLS version that is acceptable.
If not set, TLS 1.2 will be used. (optional)
#### `max_version` (_string_)

MaxVersion sets the maximum TLS version that is acceptable.
If not set, refer to crypto/tls for defaults. (optional)
#### `reload_interval` (_[Duration](#duration)_)

ReloadInterval specifies the duration after which the certificate will be reloaded
If not set, it will never be reloaded (optional)


### Tenant



Tenant is the Schema for the tenants API

_Appears in:_
- [TenantList](#tenantlist)

<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `Tenant` </b>
#### `metadata` (_[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `spec` (_[TenantSpec](#tenantspec)_)




### TenantList



TenantList contains a list of Tenant



<b> `apiVersion` _string_ </b><b> `telemetry.kube-logging.dev/v1alpha1`</b>

<b> `kind` _string_ </b><b> `TenantList` </b>
#### `metadata` (_[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#listmeta-v1-meta)_)

Refer to Kubernetes API documentation for fields of `metadata`.
#### `items` (_[Tenant](#tenant) array_)




### TenantSpec



TenantSpec defines the desired state of Tenant

_Appears in:_
- [Tenant](#tenant)

#### `subscriptionNamespaceSelectors` (_[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta) array_)


#### `logSourceNamespaceSelectors` (_[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta) array_)


#### `transform` (_[Transform](#transform)_)


#### `routeConfig` (_[RouteConfig](#routeconfig)_)






### TimeoutSettings





_Appears in:_
- [OTLPGRPC](#otlpgrpc)

#### `timeout` (_[Duration](#duration)_)

Timeout is the timeout for every attempt to send data to the backend.
A zero timeout means no timeout.


### Transform



Transform represents the Transform processor, which modifies telemetry based on its configuration

_Appears in:_
- [TenantSpec](#tenantspec)

#### `name` (_string_)

Name of the Transform processor
#### `errorMode` (_string_)

ErrorMode specifies how errors are handled while processing a statement
vaid options are: ignore, silent, propagate; (default: propagate)
#### `traceStatements` (_[TransformStatement](#transformstatement) array_)


#### `metricStatements` (_[TransformStatement](#transformstatement) array_)


#### `logStatements` (_[TransformStatement](#transformstatement) array_)




### TransformStatement



TransformStatement represents a single statement in a Transform processor
ref: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor

_Appears in:_
- [Transform](#transform)

#### `context` (_string_)


#### `conditions` (_string array_)


#### `statements` (_string array_)




