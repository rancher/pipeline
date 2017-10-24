package scheduler

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/robfig/cron"
)

type CronRunner struct {
	PipelineId string
	Cron       *cron.Cron
	Spec       string
	Timezone   string
}

func NewCronRunner(pipelineId string, spec string, timezone string) *CronRunner {
	//use Local as default
	loc, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" || timezone == "Local" {
		loc = time.Local
		if err != nil {
			logrus.Errorf("Failed to load time zone %v: %+v,use local timezone instead", timezone, err)
		}
	}
	var c *cron.Cron
	//use local timezone as default and when timezone invalid
	logrus.Debugf("newcron debug:%v,%v,%v", err, timezone, loc.String())
	c = cron.NewWithLocation(loc)

	logrus.Debugf("cron timezone is %v", c.Location().String())
	return &CronRunner{
		PipelineId: pipelineId,
		Spec:       "0 " + spec, //accept standard cron spec and convert to 6 entries for corn library
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
