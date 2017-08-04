package restfulserver

import (
	"github.com/rancher/pipeline/pipeline"
	"github.com/rancher/pipeline/scheduler"
)

var testAgent *Agent

func init() {
	testAgent = &Agent{
		connHolders:           make(map[*ConnHolder]bool),
		register:              make(chan *ConnHolder),
		unregister:            make(chan *ConnHolder),
		broadcast:             make(chan []byte),
		activityWatchlist:     make(map[string]*pipeline.Activity),
		watchActivityC:        make(chan *pipeline.Activity),
		ReWatch:               make(chan bool),
		cronRunners:           make(map[string]*scheduler.CronRunner),
		registerCronRunnerC:   make(chan *scheduler.CronRunner),
		unregisterCronRunnerC: make(chan string),
	}

}

/*
func runScheduler() {
	for {
		select {
		case cr := <-testAgent.registerCronRunnerC:
			testAgent.registerCronRunner(cr)
			return
		case pId := <-testAgent.unregisterCronRunnerC:
			testAgent.unregisterCronRunner(pId)
			return
		}
	}
}
*/
