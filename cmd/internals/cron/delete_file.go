package cron

import (
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/seaweedfs"
)

func DeleteFile() {
	list, err := arango.GetMarkedDeleteFileList(10000, 0)
	if err != nil {
		return
	}

	for _, f := range list {
		_ = arango.DeleteMarkedFileMetadata(f.Id)
		_ = seaweedfs.DeleteFile(f.Fid)
	}
}
