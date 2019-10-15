module github.com/GoogleCloudPlatform/oss-test-infra

go 1.13

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
)

require k8s.io/test-infra v0.0.0-20191015185209-be3e9cab1938
