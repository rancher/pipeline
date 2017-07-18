package scheduler

import (
	"github.com/robfig/cron"
)

type CronRunner struct {
	PipelineId string
	Cron       *cron.Cron
	Spec       string
}

func NewCronRunner(pipelineId string, spec string) *CronRunner {
	c := cron.New()
	//c.AddFunc(spec, func() { fmt.Println("run cron job one time") })
	return &CronRunner{
		PipelineId: pipelineId,
		Spec:       spec,
		Cron:       c,
	}

}

func (c *CronRunner) Start() {
	c.Cron.Start()
}

func (c *CronRunner) AddFunc(spec string, cmd func()) error {
	return c.Cron.AddFunc(spec, cmd)
}

func (c *CronRunner) Stop() {
	c.Cron.Stop()
}
