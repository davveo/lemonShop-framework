package pool

type PoolCfg struct {
	CommonGoRoutinePoolSize             int `yaml:"commonGoRoutinePoolSize"`
	CommonGoRoutinePoolMinuteExpire     int `yaml:"commonGoRoutinePoolMinuteExpire"`
	CommonGoRoutinePoolMaxBlockingTasks int `yaml:"commonGoRoutinePoolMaxBlockingTasks"`
}
