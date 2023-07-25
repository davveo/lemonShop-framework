package db

import "time"

type MysqlIns struct {
	Addr string `yaml:"addr"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
	Name string `yaml:"name"`
}

type MysqlBase struct {
	OpenSlaveRead   bool          `yaml:"openSlaveRead"`
	MaxOpenConn     int           `yaml:"maxOpenConn"`
	MaxIdleConn     int           `yaml:"maxIdleConn"`
	ConnMaxLifeTime time.Duration `yaml:"connMaxLifeTime"`
}

type MysqlCfg struct {
	Base  MysqlBase `yaml:"base"`
	Read  MysqlIns  `yaml:"read"`
	Write MysqlIns  `yaml:"write"`
}
