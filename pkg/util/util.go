package util

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"
)

func JsonToMap(val string) map[string]any {
	output := map[string]any{}
	json.Unmarshal([]byte(val), &output)
	return output
}

func JsonListToMapList(val string) []map[string]any {
	output := []map[string]any{}
	json.Unmarshal([]byte(val), &output)
	return output
}

func StringSplit(value string, delimeter string) []string {
	return strings.Split(value, delimeter)
}

func StringJoin(values []string, delimeter string) string {
	return strings.Join(values, delimeter)
}

func StringReplaceFirst(value, old, new string) string {
	return strings.Replace(value, old, new, 1)
}

func StringReplace(value, old, new string) string {
	return strings.ReplaceAll(value, old, new)
}

/*
Generates random integer between from and to
*/
func RandomNumber(from int, to int) int {
	rand.Seed(time.Now().UnixNano())
	return from + rand.Intn(to-from)
}

func RecordStartTime() int64 {
	return time.Now().UnixMilli()
}

// latency = end - start in ms
func RecordEndTime(start int64) int64 {
	return time.Now().Sub(time.UnixMilli(start)).Milliseconds()
}

func BuildLoadTestResult(isSuccess string, statusCode string, latencyMs string, response string) string {
	s, _ := json.Marshal(map[string]string{
		"isSuccess":  isSuccess,
		"statusCode": statusCode,
		"latencyMs":  latencyMs,
		"response":   response,
	})
	return string(s)
}

func CallPeriodic(interval time.Duration, f func(), quit chan bool) {

	for {
		select {
		case <-quit:
			return
		default:
			f()
			time.Sleep(interval)
		}
	}
}
