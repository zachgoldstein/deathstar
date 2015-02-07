package lib

import (
	"os"
	"io/ioutil"
)

func DoReqDiff() {

	/*
	Pseudocode for flow
	Digest command line params & config json file to generate http.Request
	Setup a spawner, which will initiate requests on a channel at a specific rate
	Setup a pool of executors, according to the concurrency, which issue the requests
	Setup an accumulator, which receives all responses and stores their stats
	Create a channel for the spawner, executor pool and accumulator to use.

	Setup an analyser, which periodically scans the accumulator and generates meaningful aggregated stats
	Setup a reporter, which renders the aggregated stats to stdOut and a live-updating page.
	 */

	reqOpts, outOpts, err := digestOptions()
	if (err != nil) {
		issueError(err)
	}


}

//issueError will print an error to stdOut that is better formatted than a normal panic
func issueError(err error) {
	panic(err)
}
