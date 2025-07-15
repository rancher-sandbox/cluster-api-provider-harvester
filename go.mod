module github.com/rancher-sandbox/cluster-api-provider-harvester

go 1.24.0

toolchain go1.24.4

require (
	github.com/containernetworking/cni v1.3.0
	github.com/containernetworking/plugins v1.7.1
	github.com/go-logr/logr v1.4.3
	github.com/harvester/harvester v1.3.2
	github.com/harvester/harvester-load-balancer v1.5.1
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.7.7
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0
	github.com/longhorn/longhorn-manager v1.9.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.37.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.83.0
	github.com/rancher/rancher/pkg/apis v0.0.0
	github.com/rancher/system-upgrade-controller/pkg/apis v0.0.0-20250701000733-99a03a0d61aa
	github.com/zach-klippenstein/goregen v0.0.0-20160303162051-795b5e3961ea
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.33.2
	k8s.io/apimachinery v0.33.2
	k8s.io/client-go v12.0.0+incompatible
	kubevirt.io/api v1.5.2
	sigs.k8s.io/cluster-api v1.10.3
	sigs.k8s.io/controller-runtime v0.21.0
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8
	sigs.k8s.io/yaml v1.5.0
)

require (
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/pkg/errors v0.9.1
	go.opentelemetry.io/otel v1.35.0 // indirect
	go.opentelemetry.io/otel/sdk v1.35.0 // indirect
	go.opentelemetry.io/otel/trace v1.35.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250218202821-56aae31c358a // indirect
)

require (
	cel.dev/expr v0.20.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coreos/go-iptables v0.8.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch v5.9.11+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fsnotify/fsnotify v1.8.0 // indirect
	github.com/fxamacker/cbor/v2 v2.8.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gxui v0.0.0-20151028112939-f85e0a97b3a4 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kubereboot/kured v1.13.1 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/openshift/custom-resource-status v1.1.2 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rancher/aks-operator v1.12.0-rc.2 // indirect
	github.com/rancher/eks-operator v1.12.0-rc.2 // indirect
	github.com/rancher/fleet/pkg/apis v0.13.0-beta.2 // indirect
	github.com/rancher/gke-operator v1.12.0-rc.2 // indirect
	github.com/rancher/lasso v0.2.3-rc3 // indirect
	github.com/rancher/norman v0.6.1 // indirect
	github.com/rancher/rke v1.8.0-rc.4 // indirect
	github.com/rancher/wrangler v1.1.2 // indirect
	github.com/rancher/wrangler/v3 v3.2.2-rc.3 // indirect
	github.com/safchain/ethtool v0.5.10 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/smartystreets/goconvey v1.8.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/vishvananda/netlink v1.3.1-0.20250303224720-0e7078ed04c8 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.3 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.33.1 // indirect
	k8s.io/apiserver v0.33.1 // indirect
	k8s.io/component-base v0.33.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250318190949-c8a335a9a2ff // indirect
	k8s.io/kubernetes v1.33.1 // indirect
	k8s.io/utils v0.0.0-20250502105355-0f33e8f1c979 // indirect
	kubevirt.io/containerized-data-importer-api v1.60.3-0.20241105012228-50fbed985de9 // indirect
	kubevirt.io/controller-lifecycle-operator-sdk/api v0.0.0-20220329064328-f3cc58c6ed90 // indirect
	sigs.k8s.io/knftables v0.0.18 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.7.0 // indirect
)

replace (
	github.com/rancher/rancher/pkg/apis => github.com/rancher/rancher/pkg/apis v0.0.0-20240919204204-3da2ae0cabd1
	github.com/rancher/rancher/pkg/client => github.com/rancher/rancher/pkg/client v0.0.0-20240919204204-3da2ae0cabd1
)

replace k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20250701173324-9bd5c66d9911

replace github.com/harvester/harvester => github.com/harvester/harvester v1.4.0-dev-20240719

replace k8s.io/client-go => k8s.io/client-go v0.33.2
