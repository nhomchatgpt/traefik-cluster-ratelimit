package redis

import (
	"fmt"
	"time"
)

type ScriptWithBreaker struct {
	script           Script
	errorCount       int64
	nextAttempt      time.Time
	breakerThreshold int64
	reattemptPeriod  int64
}

func NewScriptWithBreaker(script Script, breakerThreshold int64, reattemptPeriod int64) Script {
	return &ScriptWithBreaker{
		script:           script,
		errorCount:       0,
		nextAttempt:      time.Now(),
		breakerThreshold: breakerThreshold,
		reattemptPeriod:  reattemptPeriod,
	}
}

func (swb *ScriptWithBreaker) Run(keys []string, args ...interface{}) (interface{}, error) {
	if swb.errorCount < 3 || time.Now().After(swb.nextAttempt) {
		res, err := swb.script.Run(keys, args...)

		if err != nil {
			swb.errorCount++
			if swb.errorCount == swb.breakerThreshold {
				swb.nextAttempt = time.Now().Add(time.Duration(swb.reattemptPeriod) * time.Second)
			}
		} else {
			swb.errorCount = 0
		}

		return res, err
	} else {
		return nil, fmt.Errorf("breaker opened")
	}
}
