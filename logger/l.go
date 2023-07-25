package logger

import (
	"fmt"
	"github.com/davveo/lemonShop-framework/timeutil"
	"go.uber.org/zap"
	"os"
)

var GLogger *zap.Logger

func Init(cfg *LogCfg) (*zap.Logger, error) {
	lg, err := NewJSONLogger(
		WithDisableConsole(),
		WithField("domain", fmt.Sprintf("%s[%s]",
			cfg.AppName, os.Getenv("active"))),
		WithTimeLayout(timeutil.CSTLayout),
		WithFileP(cfg.LogSavePath),
	)
	if err != nil {
		return nil, err
	}
	GLogger = lg
	return lg, nil
}
