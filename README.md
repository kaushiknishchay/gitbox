# git on web


## Development


### Live Reload

- Install air 

  https://github.com/cosmtrek/air

  `go get -u github.com/cosmtrek/air`

- Run

  `air -d`

### Tests

- Run the below command to run all the test
  
  `go test ./...`

### Benchmark

- Run the below command

  `go test -bench=. -benchmem`

### Profiling

- Setup pprof routes

	`https://github.com/gin-contrib/pprof`

- Ensure you have Graphviz and dot installed, these are dependencies required by pprof

	`go tool pprof -http=localhost:8888 http://localhost:9090/debug/pprof/heap`
