package cronjob

import (
	"time"

	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"

	"github.com/robfig/cron/v3"
)

type ResetTrafficJob struct {
	service.ClientService
	service.ConfigService
	service.SettingService
	schedule cron.Schedule
}

func NewResetTrafficJob(schedule cron.Schedule) *ResetTrafficJob {
	return &ResetTrafficJob{schedule: schedule}
}

func (s *ResetTrafficJob) Run() {
	loc, err := s.SettingService.GetTimeLocation()
	if err != nil {
		logger.Warning("ResetTrafficJob: get time location failed: ", err)
		return
	}
	now := time.Now().In(loc)

	last, err := s.SettingService.GetGlobalResetLast()
	if err != nil {
		logger.Warning("ResetTrafficJob: get last reset time failed: ", err)
		return
	}
	// Logged, not a silent return: the setting holds the *next* boundary
	// despite its name and is not cleared when the schedule changes, so a
	// monthly-to-daily switch can look dead for up to a month.
	if last > now.Unix() {
		logger.Debug("ResetTrafficJob: next reset is at ", time.Unix(last, 0).In(loc).Format(time.RFC3339), ", nothing to do")
		return
	}

	if err = s.ClientService.ResetAllClientsTraffic(); err != nil {
		logger.Warning("ResetTrafficJob: reset all clients failed: ", err)
		return
	}

	// Restart before the bookkeeping write: clients are re-enabled in the
	// database but the core still holds the old user list, and a failed write
	// used to return early and leave them disconnected. The watchdog does not
	// help -- it only starts a core that is not running.
	if err = s.ConfigService.RestartCore(); err != nil {
		logger.Error("ResetTrafficJob: unable to restart core: ", err)
	}

	// Advance to the next boundary. schedule.Next returns the nearest upcoming
	// occurrence, so if several periods were missed (e.g. downtime) it snaps
	// forward instead of resetting once per missed period.
	next := s.schedule.Next(now)
	if err = s.SettingService.SetGlobalResetLast(next.Unix()); err != nil {
		logger.Warning("ResetTrafficJob: set last reset time failed: ", err)
		return
	}
	logger.Info("ResetTrafficJob: traffic reset for all clients; next reset at ", next.Format(time.RFC3339))
}
