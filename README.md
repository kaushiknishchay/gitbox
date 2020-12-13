# git on web

[![Go Report Card](https://goreportcard.com/badge/github.com/kaushiknishchay/gitbox)](https://goreportcard.com/report/github.com/kaushiknishchay/gitbox)

![GitHub Workflow Status](https://img.shields.io/github/workflow/status/kaushiknishchay/gitbox/Go?label=Build)

![GitHub Release Date](https://img.shields.io/github/release-date/kaushiknishchay/gitbox?label=Last%20release%20date) ![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kaushiknishchay/gitbox)




![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/kaushiknishchay/gitbox)
---


## Deployed Version

https://git-on-web.herokuapp.com/



---

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
