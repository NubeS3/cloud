package cron

import (
	"github.com/robfig/cron/v3"
)

var c *cron.Cron

func init() {
	c = cron.New()

	println("initialize cron jobs")
	//_, _ = c.AddFunc("@daily", DeleteFile)
	_, _ = c.AddFunc("@daily", DeleteFile)
	c.Start()
}

func CleanUp() {
	c.Stop()
}
