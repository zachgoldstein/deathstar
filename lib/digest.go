package lib

type RequestOptions struct {
	Method string
	URL string
	Headers map[string]string
	CPUs uint
}

type OutputOptions struct {
	OutputHTML bool
	ShowFullJSON bool
	HTMLOutputLocation string
	JSONSchema string
}

type Options struct {
	Rate uint
}

// digestOptions will combine command line options and the config json file to create the options objects
func digestOptions()(reqOpts RequestOptions, outOpts OutputOptions, err error) {
	return RequestOptions{
		Method : "GET",
		URL : "localhost:8080/test",
	}, OutputOptions {
		OutputHTML : true,
		ShowFullJSON : true,
	}, nil
}
