package lib

//Spawner is responsible for initiating requests on a channel at a specific rate
//It manages a pool of executors that will create and issue requests
type Spawner struct {
	Rate uint
	ExecutorPool []*Executor
}

func NewSpawner() *Spawner {

}

func (s *Spawner) spawn (opts Options) {

}
