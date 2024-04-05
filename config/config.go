package config

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	_ "github.com/spf13/viper/remote"
)

func initialiseRemote(v *viper.Viper) error {
	consulUrl := os.Getenv("CONSUL_URL")
	_ = v.AddRemoteProvider("consul", consulUrl, "WA_MULTI_SESSION")
	v.SetConfigType("yaml")
	return v.ReadRemoteConfig()
}

func initialiseFileAndEnv(v *viper.Viper, env string) error {
	v.SetConfigName(configName[env])
	for _, path := range searchPath {
		v.AddConfigPath(path)
	}
	v.SetEnvPrefix("WA_MULTI_SESSION")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return v.ReadInConfig()
}

func initialiseDefaults(v *viper.Viper) {
	for key, value := range configDefaults {
		v.SetDefault(key, value)
	}
}

func Initialize() {
	v := viper.New()
	initialiseDefaults(v)
	if err := initialiseRemote(v); err != nil {
		log.Warningf("No remote server configured will load configuration from file and environment variables: %+v", err)
		if err := initialiseFileAndEnv(v, Env); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				log.Warning("No 'config.yaml' file found on search paths. Will either use environment variables or defaults")
			} else {
				log.Fatalf("Error occurred during loading config: %s", err.Error())
			}
		}
	}

	err := v.Unmarshal(&Conf)
	if err != nil {
		log.Printf("Error un-marshalling configuration: %s", err.Error())
	}
}
