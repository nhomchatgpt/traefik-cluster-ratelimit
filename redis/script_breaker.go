package redis

import (
	"fmt"
	"time"
)

type ScriptWithBreaker struct {
	script      Script
	errorCount  int
	nextAttempt time.Time
}

func NewScriptWithBreaker(script Script) Script {
	return &ScriptWithBreaker{
		script:      script,
		errorCount:  0,
		nextAttempt: time.Now(),
	}
}

func (swb *ScriptWithBreaker) Run(keys []string, args ...interface{}) (interface{}, error) {
	if swb.errorCount < 3 || time.Now().After(swb.nextAttempt) {
		res, err := swb.script.Run(keys, args...)

		if err != nil {
			swb.errorCount++
			if swb.errorCount == 3 {
				swb.nextAttempt = time.Now().Add(15 * time.Second)
			}
		} else {
			swb.errorCount = 0
		}

		return res, err
	} else {
		return nil, fmt.Errorf("breaker opened")
	}
}
