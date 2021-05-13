package cron

import (
	"github.com/robfig/cron/v3"
)

var c *cron.Cron

func init() {
	c = cron.New()

	_, _ = c.AddFunc("@daily", DeleteFile)
}

func CleanUp() {
	c.Stop()
}
