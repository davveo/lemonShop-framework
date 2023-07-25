package cache

type RedisCfg struct {
	Addr        string `yaml:"addr"`
	Db          int    `yaml:"db"`
	MaxRetries  int    `yaml:"maxRetries"`
	MinIdleConn int    `yaml:"minIdleConn"`
	Pass        string `yaml:"pass"`
	PoolSize    int    `yaml:"poolSize"`
}
