package redis

import (
	"fmt"
	"strings"
)

type RedisScript struct {
	client    *RedisClient
	script    string
	scriptSha string
}

func (r *RedisClient) NewScript(script string) *RedisScript {
	rs := &RedisScript{
		client:    r,
		script:    script,
		scriptSha: "",
	}

	return rs
}

func (rs *RedisScript) Run(keys []string, args ...interface{}) (string, error) {
	conn, err := rs.client.Get()
	if err != nil {
		return "", err
	}
	defer rs.client.Put(conn)

	argsarray, err := convertToStringArray(args)
	if err != nil {
		return "", err
	}

	params := []string{"EVALSHA", rs.scriptSha}
	params = append(params, fmt.Sprintf("%d", len(keys)))
	params = append(params, keys...)
	params = append(params, argsarray...)

	present := false
	if rs.scriptSha != "" {
		res, err := sendCommand(conn, params...)
		if err != nil {
			return "", err
		}
		if res.Success == RESP_FAIL || strings.HasPrefix(res.Result, "NOSCRIPT") {
			present = false
		} else {
			return res.Result, nil
		}
	}

	// the script was not present, let's load it and runs it
	if !present {
		res, err := sendCommand(conn, "SCRIPT", "LOAD", rs.script)
		if err != nil {
			return "", err
		}
		// sha
		if len(res.Result) == 40 {
			rs.scriptSha = res.Result
		} else {
			return "", fmt.Errorf("not able to load script: %s", res.Result)
		}
	}

	params = []string{"EVALSHA", rs.scriptSha}
	params = append(params, fmt.Sprintf("%d", len(keys)))
	params = append(params, keys...)
	params = append(params, argsarray...)

	// run the script
	res, err := sendCommand(conn, params...)
	if err != nil {
		return "", err
	}
	if res.Success == RESP_FAIL || strings.HasPrefix(res.Result, "NOSCRIPT") {
		return "", fmt.Errorf("not able to run the script: %s", res.Result)
	} else {
		return res.Result, nil
	}
}

func convertToStringArray(args ...interface{}) ([]string, error) {
	// Initialize a slice to hold the string values
	result := make([]string, len(args))

	// Loop through the interface slice and convert each element to a string
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			result[i] = v
		default:
			// Convert non-string types to string using fmt.Sprintf
			result[i] = fmt.Sprintf("%v", arg)
		}
	}

	return result, nil
}
