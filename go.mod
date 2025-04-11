module github.com/kube-logging/telemetry-controller

go 1.24.0

require (
	emperror.dev/errors v0.8.1
	github.com/cisco-open/operator-tools v0.37.0
	github.com/go-logr/logr v1.4.2
	github.com/google/go-cmp v0.7.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/mitchellh/mapstructure v1.5.0
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.37.0
	github.com/open-telemetry/opentelemetry-operator v0.121.0
	github.com/prometheus/client_golang v1.22.0
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/collector/component v1.29.0
	go.opentelemetry.io/collector/config/configauth v0.123.0
	go.opentelemetry.io/collector/config/configcompression v1.29.0
	go.opentelemetry.io/collector/config/configopaque v1.29.0
	go.opentelemetry.io/collector/config/configtelemetry v0.123.0
	go.opentelemetry.io/collector/otelcol v0.123.0
	go.opentelemetry.io/collector/pipeline v0.123.0
	go.opentelemetry.io/collector/service v0.123.0
	go.uber.org/zap v1.27.0
	k8s.io/api v0.32.3
	k8s.io/apimachinery v0.32.3
	k8s.io/client-go v0.32.3
	sigs.k8s.io/controller-runtime v0.20.4
	sigs.k8s.io/yaml v1.4.0
)

require (
	cel.dev/expr v0.19.1 // indirect
	dario.cat/mergo v1.0.1 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/briandowns/spinner v1.23.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cisco-open/k8s-objectmatcher v1.10.0 // indirect
	github.com/cppforlife/go-patch v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/ebitengine/purego v0.8.2 // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/cel-go v0.22.0 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/iancoleman/orderedmap v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.2 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.63.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/shirou/gopsutil/v4 v4.25.1 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/tklauser/go-sysconf v0.3.13 // indirect
	github.com/tklauser/numcpus v0.7.0 // indirect
	github.com/wayneashleyberry/terminal-dimensions v1.1.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.123.0 // indirect
	go.opentelemetry.io/collector/component/componenttest v0.123.0 // indirect
	go.opentelemetry.io/collector/confmap v1.29.0 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.123.0 // indirect
	go.opentelemetry.io/collector/connector v0.123.0 // indirect
	go.opentelemetry.io/collector/connector/connectortest v0.123.0 // indirect
	go.opentelemetry.io/collector/connector/xconnector v0.123.0 // indirect
	go.opentelemetry.io/collector/consumer v1.29.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.123.0 // indirect
	go.opentelemetry.io/collector/consumer/consumertest v0.123.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.123.0 // indirect
	go.opentelemetry.io/collector/exporter v0.123.0 // indirect
	go.opentelemetry.io/collector/exporter/exportertest v0.123.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.123.0 // indirect
	go.opentelemetry.io/collector/extension v1.29.0 // indirect
	go.opentelemetry.io/collector/extension/extensionauth v1.29.0 // indirect
	go.opentelemetry.io/collector/extension/extensioncapabilities v0.123.0 // indirect
	go.opentelemetry.io/collector/extension/extensiontest v0.123.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.29.0 // indirect
	go.opentelemetry.io/collector/internal/fanoutconsumer v0.123.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.123.0 // indirect
	go.opentelemetry.io/collector/pdata v1.29.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.123.0 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.123.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.123.0 // indirect
	go.opentelemetry.io/collector/processor v1.29.0 // indirect
	go.opentelemetry.io/collector/processor/processortest v0.123.0 // indirect
	go.opentelemetry.io/collector/processor/xprocessor v0.123.0 // indirect
	go.opentelemetry.io/collector/receiver v1.29.0 // indirect
	go.opentelemetry.io/collector/receiver/receivertest v0.123.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.123.0 // indirect
	go.opentelemetry.io/collector/semconv v0.123.0 // indirect
	go.opentelemetry.io/collector/service/hostcapabilities v0.123.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.10.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.15.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.35.0 // indirect
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.11.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.11.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.57.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.11.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.35.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.35.0 // indirect
	go.opentelemetry.io/otel/log v0.11.0 // indirect
	go.opentelemetry.io/otel/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.11.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20250218142911-aa4b98e5adaa // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/oauth2 v0.28.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/term v0.30.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250303144028-a0af3efb3deb // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250227231956-55c901821b1e // indirect
	google.golang.org/grpc v1.71.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.32.3 // indirect
	k8s.io/apiserver v0.32.3 // indirect
	k8s.io/component-base v0.32.3 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20241212222426-2c72e554b1e7 // indirect
	k8s.io/utils v0.0.0-20241210054802-24370beab758 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.31.0 // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.5.0 // indirect
)

// v1.51.0 is present in the go.mod file of the opentelemetry-operator as retracted
// automated dependency updates can fail because of this.
exclude github.com/open-telemetry/opentelemetry-operator v1.51.0
