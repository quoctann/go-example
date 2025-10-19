package main

import (
	"go-example/command"
	"go-example/cronjob"
	"go-example/redis"
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
}
