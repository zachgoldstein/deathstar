package lib

import "fmt"

var CurrentLogType = "all"

//Log maps fmt.PrintLn, but with a categorization to customize logging results
func Log(logType string, toPrint ...interface{}) {
	if (CurrentLogType == logType || logType == "all" || logType == "temp") {
		fmt.Print(toPrint)
	}
}
