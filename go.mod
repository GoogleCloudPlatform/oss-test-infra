module github.com/GoogleCloudPlatform/oss-test-infra

go 1.16

// Pin all k8s.io staging repositories to kubernetes v0.18.6
// When bumping Kubernetes dependencies, you should update each of these lines
// to point to the same kubernetes v0.KubernetesMinor.KubernetesPatch version
// before running update-deps.sh.
replace (
	cloud.google.com/go/pubsub => cloud.google.com/go/pubsub v1.3.1
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/golang/lint => golang.org/x/lint v0.0.0-20190301231843-5614ed5bae6f
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1

	// Upstream is unmaintained. This fork introduces two important changes:
	// * We log an error if writing a cache key fails (e.G. because disk is full)
	// * We inject a header that allows ghproxy to detect if the response was revalidated or a cache miss
	github.com/gregjones/httpcache => github.com/alvaroaleman/httpcache v0.0.0-20210618195546-ab9a1a3f8a38

	golang.org/x/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20190709130402-674ba3eaed22
	k8s.io/api => k8s.io/api v0.21.3
	k8s.io/client-go => k8s.io/client-go v0.21.1

	k8s.io/test-infra => k8s.io/test-infra v0.0.0-20211014223256-1433e96a1f3d
)

require (
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/test-infra v0.0.0-20211014223256-1433e96a1f3d
)
