package lib

func DoScaleTest() {

	reqOpts, outOpts, err := digestOptions()
	if (err != nil) {
		panic(err)
	}

	choreographer := NewChoreographer(reqOpts, outOpts)
	choreographer.Start()

}
