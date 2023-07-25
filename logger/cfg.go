package logger

type LogCfg struct {
	AppName     string `yaml:"appName"`
	LogSavePath string `yaml:"logSavePath"`
	TimeFormat  string `yaml:"timeFormat"`
}
