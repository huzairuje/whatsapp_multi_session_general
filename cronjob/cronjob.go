package cronjob

import (
	"fmt"
	"whatsapp_multi_session_general/commandhandler"
	"whatsapp_multi_session_general/config"
	"whatsapp_multi_session_general/cronjob/crontab"

	"go.mau.fi/whatsmeow/types"
)

type CronJobs struct {
	CommandHandler commandhandler.CommandHandler
}

func NewCronJobs(commandhandler commandhandler.CommandHandler) *CronJobs {
	return &CronJobs{
		CommandHandler: commandhandler,
	}
}

func (c CronJobs) Run() {
	//initiate crontab
	crontabInit := crontab.New()

	//add jobs here based on the configuration
	if config.Conf.Cronjob.AutoPresence.Enable {
		// Add job and print the errors
		schedule := config.Conf.Cronjob.AutoPresence.CronJobSchedule
		if schedule == "" {
			schedule = "* * * * *"
		}
		err := crontabInit.AddJob(schedule, func() {
			err := c.AutoPresence()
			if err != nil {
				fmt.Errorf("err on job AutoPresence : %v ", err)
				return
			}
		})
		if err != nil {
			fmt.Errorf("err on job AutoPresence : %v ", err)
			crontabInit.Shutdown()
		}
	}
}

func (c CronJobs) AutoPresence() (err error) {
	devices := commandhandler.Clients
	if len(devices) > 0 {
		for _, val := range devices {
			if val.Store.ID.User != "" {
				// send presence
				if val.Store.ID != nil {
					err = val.SendPresence(types.PresenceAvailable)
					if err != nil {
						fmt.Errorf("err client.SendPresence : %v ", err)
					}
					fmt.Println("send presence is success")
				}
			} else {
				continue
			}
		}
	}
	return
}
