package lib

import (
	"log"
)

//Log maps fmt.PrintLn, but with a categorization to customize logging results
func Log(logType string, toPrint ...interface{}) {
	if showLogs && containsSubstring(logType) {
		log.Print(toPrint...)
	}
}

var acceptableLogTypes = []string{}
//var acceptableLogTypes = []string{"all", "temp", "analyse", "top", "spawn"}
var showLogs = true

func containsSubstring(logType string) bool {
	for _, acceptableLogType := range acceptableLogTypes {
		if (logType == acceptableLogType) {
			return true
		}
	}
	return false
}
