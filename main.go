package main

import (
	"go-example/command"
	"go-example/cronjob"
	"go-example/redis"
	"go-example/streaming"
	"go-example/websocket"
)

func main() {
	skip := true

	cronjob.RunNotUseLibCronJobExample(skip)
	cronjob.RunUseLibCronJobExample(skip)
	cronjob.RunCronRedisLock(skip)

	redis.RunExample(skip)

	command.RunBasic(skip)
	command.RunCobraCLI(skip)

	websocket.RunExample(skip)

	streaming.RunExample(skip)
	// need update skip flag, build and run separately to test game streaming
	streaming.RunGameServerExample(skip)
	streaming.RunGameClientExample(skip)
	// video streaming (like movies, not livestream low latency)
	streaming.RunHLSExample(false)
}
