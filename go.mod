module github.com/gardener/test-infra

go 1.13

require (
	cloud.google.com/go v0.45.1
	github.com/Masterminds/semver v1.4.2
	github.com/argoproj/argo v2.4.0+incompatible
	github.com/bradleyfalzon/ghinstallation v0.1.2
	github.com/gardener/gardener v0.0.0-20191114144236-0e90aba93f3d
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-openapi/spec v0.19.4
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/golang/mock v1.3.1
	github.com/google/go-github/v27 v27.0.4
	github.com/googleapis/gnostic v0.3.1 // indirect
	github.com/gorilla/mux v1.6.2
	github.com/hashicorp/go-multierror v1.0.0
	github.com/joho/godotenv v1.3.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/mattn/go-runewidth v0.0.4 // indirect
	github.com/minio/minio-go v6.0.13+incompatible
	github.com/olekukonko/tablewriter v0.0.1
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1 // indirect
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4 // indirect
	github.com/prometheus/common v0.7.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.4.0
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20191119213627-4f8c1d86b1ba // indirect
	golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	golang.org/x/net v0.0.0-20191119073136-fc4aabc6c914 // indirect
	golang.org/x/sys v0.0.0-20191120155948-bd437916bb0e // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.0.1 // indirect
	google.golang.org/api v0.10.0
	google.golang.org/genproto v0.0.0-20190911173649-1774047e7e51
	gopkg.in/yaml.v2 v2.2.7
	k8s.io/api v0.0.0-20191121015604-11707872ac1c
	k8s.io/apiextensions-apiserver v0.0.0-20191121021419-88daf26ec3b8
	k8s.io/apimachinery v0.0.0-20191121015412-41065c7a8c2a
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/code-generator v0.0.0-20190831074504-732c9ca86353
	k8s.io/helm v2.14.2+incompatible
	k8s.io/kube-openapi v0.0.0-20191107075043-30be4d16710a
	k8s.io/test-infra v0.0.0-20191121090728-82f9ffaa1274
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6 // indirect
	sigs.k8s.io/controller-runtime v0.3.0
	sigs.k8s.io/yaml v1.1.0
)

replace (
	github.com/census-instrumentation/opencensus-proto => github.com/census-instrumentation/opencensus-proto v0.0.3-0.20181214143942-ba49f56771b8
	github.com/onsi/ginkgo => github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega => github.com/onsi/gomega v1.5.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.2
	k8s.io/api => k8s.io/api v0.0.0-20190313235455-40a48860b5ab //kubernetes-1.14.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190315093550-53c4693659ed // kubernetes-1.14.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190313205120-d7deff9243b1 // kubernetes-1.14.0
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190313205120-8b27c41bdbb1 // kubernetes-1.14.0
	k8s.io/client-go => k8s.io/client-go v11.0.0+incompatible // kubernetes-1.14.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20190314002537-50662da99b70 // kubernetes-1.14.0
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190311093542-50b561225d70 // kubernetes-1.14.0
	k8s.io/component-base => k8s.io/component-base v0.0.0-20190314000054-4a91899592f4 // kubernetes-1.14.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20190314000639-da8327669ac5 // kubernetes-1.14.0
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20190320154901-5e45bb682580
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.2.0-beta.4
)
