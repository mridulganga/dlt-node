package constants

import (
	"github.com/mridulganga/dlt-node/pkg/util"
	log "github.com/sirupsen/logrus"
)

var EXPOSED_METHODS = map[string]any{
	"log":    func(msg any) { log.Infof("lua: %v", msg) },
	"random": util.RandomNumber,

	// rest api calling
	"restGet":    util.RGet,
	"restPut":    util.RPut,
	"restPost":   util.RPost,
	"restPatch":  util.RPatch,
	"restDelete": util.RDelete,

	// json string and map utils
	"jsonToMap":         util.JsonToMap,
	"jsonListToMapList": util.JsonListToMapList,

	// string utils
	"stringSplit":        util.StringSplit,
	"stringJoin":         util.StringJoin,
	"stringReplaceFirst": util.StringReplaceFirst,
	"stringReplace":      util.StringReplace,

	"recordStartTime": util.RecordStartTime,
	"recordEndTime":   util.RecordEndTime,

	// result object
	"buildResult": util.BuildLoadTestResult,
}
