module github.com/rancher-sandbox/cluster-api-provider-harvester

go 1.23

toolchain go1.23.3

require (
	github.com/containernetworking/cni v1.2.3
	github.com/containernetworking/plugins v1.6.0
	github.com/harvester/harvester v1.2.1
	github.com/harvester/harvester-load-balancer v0.2.3
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.7.5
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.2.0
	github.com/longhorn/longhorn-manager v1.7.2
	github.com/onsi/ginkgo/v2 v2.22.0
	github.com/onsi/gomega v1.36.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.78.2
	github.com/rancher/rancher/pkg/apis v0.0.0
	github.com/rancher/system-upgrade-controller/pkg/apis v0.0.0-20210727200656-10b094e30007
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56
	k8s.io/api v0.31.2
	k8s.io/apimachinery v0.31.2
	k8s.io/client-go v12.0.0+incompatible
	kubevirt.io/api v0.54.0
	sigs.k8s.io/cluster-api v1.5.1
	sigs.k8s.io/controller-runtime v0.19.0
)

require (
	github.com/coreos/go-iptables v0.8.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/gxui v0.0.0-20151028112939-f85e0a97b3a4 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/rancher/eks-operator v1.1.5 // indirect
	github.com/rancher/gke-operator v1.1.4 // indirect
	github.com/safchain/ethtool v0.4.1 // indirect
	github.com/vishvananda/netlink v1.3.0 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	sigs.k8s.io/knftables v0.0.17 // indirect
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.6.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-logr/logr v1.4.2
	github.com/go-logr/zapr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/gobuffalo/flect v1.0.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20241029153458-d1b30febd7db // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/imdario/mergo v0.3.13 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/openshift/custom-resource-status v1.1.2 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rancher/aks-operator v1.0.7 // indirect
	github.com/rancher/fleet/pkg/apis v0.0.0-20230123175930-d296259590be // indirect
	github.com/rancher/lasso v0.0.0-20240705194423-b2a060d103c1 // indirect
	github.com/rancher/norman v0.0.0-20221205184727-32ef2e185b99 // indirect
	github.com/rancher/rke v1.3.18 // indirect
	github.com/rancher/wrangler v1.1.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/zach-klippenstein/goregen v0.0.0-20160303162051-795b5e3961ea
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/oauth2 v0.22.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/term v0.25.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.26.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.31.2 // indirect
	k8s.io/apiserver v0.31.1 // indirect
	k8s.io/component-base v0.31.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240228011516-70dd3763d340 // indirect
	k8s.io/utils v0.0.0-20240921022957-49e7df575cb6 // indirect
	kubevirt.io/containerized-data-importer-api v1.47.0 // indirect
	kubevirt.io/controller-lifecycle-operator-sdk/api v0.0.0-20220329064328-f3cc58c6ed90 // indirect
	sigs.k8s.io/cli-utils v0.35.0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0
)

replace (
	github.com/openshift/api => github.com/openshift/api v0.0.0-20191219222812-2987a591a72c
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/operator-framework/operator-lifecycle-manager => github.com/operator-framework/operator-lifecycle-manager v0.0.0-20190128024246-5eb7ae5bdb7a
	github.com/rancher/rancher/pkg/apis => github.com/rancher/rancher/pkg/apis v0.0.0-20230124173128-2207cfed1803
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc => go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.20.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.20.0
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/otlp => go.opentelemetry.io/otel/exporters/otlp v0.20.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v0.20.0
	go.opentelemetry.io/proto/otlp => go.opentelemetry.io/proto/otlp v0.7.0
	k8s.io/api => k8s.io/api v0.27.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.27.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.27.2
	k8s.io/apiserver => k8s.io/apiserver v0.27.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.27.2
	k8s.io/client-go => k8s.io/client-go v0.27.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.27.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.27.2
	k8s.io/code-generator => k8s.io/code-generator v0.27.2
	k8s.io/component-base => k8s.io/component-base v0.27.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.27.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.27.2
	k8s.io/cri-api => k8s.io/cri-api v0.27.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.27.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.27.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.27.2
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20230501164219-8b0f38b5fd1f
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.27.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.27.2
	k8s.io/kubectl => k8s.io/kubectl v0.27.2
	k8s.io/kubelet => k8s.io/kubelet v0.27.2
	k8s.io/kubernetes => k8s.io/kubernetes v1.26.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.27.2
	k8s.io/metrics => k8s.io/metrics v0.27.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.27.2
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.27.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.27.2
	kubevirt.io/api => github.com/kubevirt/api v0.54.0
	kubevirt.io/client-go => github.com/kubevirt/client-go v0.54.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.15.1
)
