package ultis

import "github.com/NubeS3/cloud/cmd/internals/models/nats"

type DownloadBandwidthLogger struct {
	Uid        string
	From       string
	SourceType string
}

func (logger *DownloadBandwidthLogger) Write(p []byte) (int, error) {
	err := nats.SendBandwidthLog(int64(len(p)), logger.Uid, logger.From, logger.SourceType)
	return len(p), err
}
