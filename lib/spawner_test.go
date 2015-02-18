package lib

import (
	"testing"
	c "github.com/smartystreets/goconvey/convey"
	"time"
	"github.com/nicholasf/fakepoint"
)

func TestMakeRequests(t *testing.T) {
	c.Convey("With a working spawner instance", t, func(){
		responseStatsChan := make(chan ResponseStats)
		overallStatsChan := make(chan OverallStats)
		rate := 3
		spawner := NewSpawner(rate, time.Second * 2, responseStatsChan, overallStatsChan, DefaultRequestOptions)
		c.Convey("I can trigger the spawner to trigger the correct number of requests", func(){
			numRequests := 0
			go func() {
				for _ = range spawner.RequestChan {
					numRequests += 1
				}
			}()
			spawner.MakeRequests()
			c.So(numRequests, c.ShouldEqual, rate)
		})
	})
}

func TestTimeouts(t *testing.T) {
	c.Convey("With a working spawner instance", t, func(){
		responseStatsChan := make(chan ResponseStats)
		overallStatsChan := make(chan OverallStats)
		timeout := time.Millisecond * 10
		spawner := NewSpawner(0, timeout, responseStatsChan, overallStatsChan, DefaultRequestOptions)
		c.Convey("It will timeout at the correct time", func(){
			spawner.Start()
			done := false
			go func() {
				for _ = range spawner.Done {
					done = true
				}
			}()
			time.Sleep(time.Millisecond * 30)
			c.So(done, c.ShouldEqual, true)
		})
	})
}

func TestPeriodicRequests(t *testing.T) {
	c.Convey("With a working spawner instance", t, func(){
		responseStatsChan := make(chan ResponseStats)
		overallStatsChan := make(chan OverallStats)
		rate := 3
		reqOpts := RequestOptions{
			URL : "http://fake.com",
			Method : "GET",
			Payload : []byte(""),
		}
		spawner := NewSpawner(rate, time.Second * 2, responseStatsChan, overallStatsChan, reqOpts)
		c.Convey("The spawner the correct number of requests periodically", func(){

			accumulator := NewAccumulator(spawner.StatsChan, overallStatsChan)

			maker := fakepoint.NewFakepointMaker()
			maker.NewGet(reqOpts.URL, 200)
			spawner.CustomClient = maker.Client()

			go spawner.Start()

			time.Sleep(time.Second * 2)
			c.So(len(accumulator.Stats), c.ShouldEqual, rate)

			time.Sleep(time.Second * 1)
			c.So(len(accumulator.Stats), c.ShouldEqual, rate * 2)
		})
	})
}

