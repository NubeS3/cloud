package cron

import (
	"github.com/NubeS3/cloud/cmd/internals/models/arango"
	"github.com/NubeS3/cloud/cmd/internals/models/nats"
	"github.com/NubeS3/cloud/cmd/internals/models/seaweedfs"
	"time"
)

func DeleteFile() {
	list, err := arango.GetMarkedDeleteFileList(10000, 0)
	if err != nil {
		return
	}

	for _, f := range list {
		err = arango.DeleteMarkedFileMetadata(f.Id)
		err = seaweedfs.DeleteFile(f.Fid)
		err = nats.SendDeleteFileEvent(f.Id, f.Fid, f.Name, f.Size, f.BucketId, time.Now(), f.Uid)
	}
}
