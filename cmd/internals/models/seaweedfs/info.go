package seaweedfs

import "github.com/linxGnu/goseaweedfs"

func SeaweedInfo() (*goseaweedfs.SystemStatus, error) {
	return sw.Status()
}

func SeaweedClusterStatus() (*goseaweedfs.ClusterStatus, error) {
	return sw.ClusterStatus()
}
