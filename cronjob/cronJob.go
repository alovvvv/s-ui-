package cronjob

import (
	"time"

	"github.com/alireza0/s-ui/logger"

	"github.com/robfig/cron/v3"
)

// cronParser accepts standard 5-field cron, optional leading seconds (6-field)
// and descriptors (@daily, @weekly, @every 10s, ...). Used both for the cron
// engine and for parsing the user-provided globalReset spec.
var cronParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

type CronJob struct {
	cron *cron.Cron
}

func NewCronJob() *CronJob {
	return &CronJob{}
}

func (c *CronJob) Start(loc *time.Location, trafficAge int, statsBucketSeconds int64, globalReset string) error {
	// Recover: robfig/cron does not recover panics by default, and gin only
	// covers the HTTP side. SkipIfStillRunning: the stats job fires every 10s,
	// can block that long on the write lock, and overlapping runs each drain
	// the core's traffic counters.
	c.cron = cron.New(
		cron.WithLocation(loc),
		cron.WithParser(cronParser),
		cron.WithChain(
			cron.Recover(cron.DefaultLogger),
			cron.SkipIfStillRunning(cron.DefaultLogger),
		),
	)

	// Registered before Start, not from a goroutine racing it.
	addJob := func(spec string, job cron.Job, name string) {
		if _, err := c.cron.AddJob(spec, job); err != nil {
			logger.Warning("unable to schedule ", name, " <", spec, ">: ", err)
		}
	}

	// Start stats job
	addJob("@every 10s", NewStatsJob(trafficAge > 0, statsBucketSeconds), "stats job")
	// Start expiry job
	addJob("@every 1m", NewDepleteJob(), "deplete job")
	// Periodic global traffic reset, only when a valid cron spec is configured
	if globalReset != "" && globalReset != "off" {
		schedule, err := cronParser.Parse(globalReset)
		if err != nil {
			logger.Warning("invalid globalReset cron spec <", globalReset, ">: ", err)
		} else {
			addJob(globalReset, NewResetTrafficJob(schedule), "traffic reset job")
		}
	}
	// Start deleting old stats
	if trafficAge > 0 {
		addJob("@daily", NewDelStatsJob(trafficAge), "old stats cleanup")
	}
	// Start core if it is not running
	addJob("@every 5s", NewCheckCoreJob(), "core watchdog")
	// database WAL checkpoint
	addJob("@every 10m", NewWALCheckpointJob(), "WAL checkpoint")

	c.cron.Start()
	return nil
}

func (c *CronJob) Stop() {
	if c.cron != nil {
		c.cron.Stop()
	}
}
