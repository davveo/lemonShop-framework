package mq

import (
	"errors"
	"sync"
	"time"
)

var (
	AckDataNil = errors.New("ack data nil")

	UtfallSecond = "2006-01-02 15:04:05"
	cstZone      = time.FixedZone("CST", 8*3600)

	statusLock sync.Mutex
	status     bool = false
)
