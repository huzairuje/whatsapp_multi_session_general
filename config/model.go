package config

var (
	Conf Config
	Env  string

	searchPath = []string{
		"/etc/whatsapp_multi_session",
		"$HOME/.wa_multi_session",
		".",
	}
	configDefaults = map[string]interface{}{
		"port":       1234,
		"logLevel":   "DEBUG",
		"logFormat":  "text",
		"signString": "supersecret",
	}
	configName = map[string]string{
		"local": "config.local",
		"dev":   "config.dev",
		"uat":   "config.uat",
		"prod":  "config.prod",
		"test":  "config.test",
	}
)

type Config struct {
	Env            string   `mapstructure:"env"`
	Port           int      `mapstructure:"port"`
	StartUp        StartUp  `mapstructure:"startUp"`
	ShutDown       ShutDown `mapstructure:"shutDown"`
	AutoLogout     bool     `mapstructure:"autoLogout"`
	AutoDisconnect bool     `mapstructure:"autoDisconnect"`
	Cronjob        Cronjob  `mapstructure:"cronjob"`
}

type StartUp struct {
	EnableAutoLogin bool `json:"enableAutoLogin"`
}

type ShutDown struct {
	EnableAutoShutDown bool `json:"enableAutoLogOut"`
}

type Cronjob struct {
	AutoPresence AutoPresence `mapstructure:"autoPresence"`
}

type AutoPresence struct {
	Enable          bool   `mapstructure:"enable"`
	CronJobSchedule string `mapstructure:"cronJobSchedule"`
}
