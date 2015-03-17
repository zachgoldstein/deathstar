FILES=./lib/*.go

fmt:
	go fmt ${FILES}

deps:
	go get github.com/smartystreets/goconvey
	go get github.com/xeipuuv/gojsonschema
	go get github.com/nicholasf/fakepoint
	go get github.com/aybabtme/uniplot/histogram
	go get github.com/jroimartin/gocui
	go get github.com/cheggaaa/pb
	go get import github.com/googollee/go-socket.io

test:
	go test ${FILES} -v

live-test:
	goconvey

doc:
	pkill godoc; godoc -http=":7080" &