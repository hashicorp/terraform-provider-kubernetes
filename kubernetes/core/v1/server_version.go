package v1

import (
	"k8s.io/client-go/kubernetes"

	gversion "github.com/hashicorp/go-version"
)

func getServerVersion(connection *kubernetes.Clientset) (*gversion.Version, error) {
	sv, err := connection.ServerVersion()
	if err != nil {
		return nil, err
	}

	return gversion.NewVersion(sv.String())
}

func serverVersionGreaterThanOrEqual(connection *kubernetes.Clientset, version string) (bool, error) {
	sv, err := getServerVersion(connection)
	if err != nil {
		return false, err
	}
	// server version that we need to compare with
	cv, err := gversion.NewVersion(version)
	if err != nil {
		return false, err
	}
	return sv.GreaterThanOrEqual(cv), nil
}
