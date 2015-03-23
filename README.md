# Deathstar

An API integration testing tool that will send a specific amount of traffic at an API, measuring it's ability to scale to specific amounts of load.

WARNING: Not fully functional yet.

## Features
- Failure is a 400/500 or invalid response based on JSON schema
- Quantile results
- Warm up period
- w/o warmup, time to hit scale
- tunable # of requests issued per second.
- cpus??
- live updating results
- pretty output (html and stdOut)

- mention Ulimit

Future: 
- request chains?
- requests with scripts in between?