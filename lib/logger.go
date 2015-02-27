package lib

import (
	"log"
)

var CurrentLogType = "all"

//Log maps fmt.PrintLn, but with a categorization to customize logging results
func Log(logType string, toPrint ...interface{}) {
	if containsSubstring(logType) {
		log.Print(toPrint...)
	}
}

var acceptableLogTypes = []string{"all", "temp", "analyse", "top", "spawn"}

func containsSubstring(logType string) bool {
	for _, acceptableLogType := range acceptableLogTypes {
		if (logType == acceptableLogType) {
			return true
		}
	}
	return false
}
