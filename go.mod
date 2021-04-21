module github.com/GoogleCloudPlatform/oss-test-infra

go 1.13

replace (
	k8s.io/client-go => k8s.io/client-go v0.20.2
)

require (
	github.com/bazelbuild/bazel-gazelle v0.18.1 // indirect
	github.com/google/go-cmp v0.5.2
	github.com/knative/build v0.3.1-0.20190330033454-38ace00371c7 // indirect
	github.com/knative/pkg v0.0.0-20190330034653-916205998db9 // indirect
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/repo-infra v0.0.0-20190921032325-1fedfadec8ce // indirect
	k8s.io/test-infra v0.0.0-20210421004810-2f8b3ae53188
)
