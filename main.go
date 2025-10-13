package main

import (
	"go-example/command"
	"go-example/cronjob"
	"go-example/redis"
)

func main() {
	cronjob.RunNotUseLibCronJobExample(true)
	cronjob.RunUseLibCronJobExample(true)
	cronjob.RunCronRedisLock(true)

	redis.RunExample(true)

	command.RunBasic(true)
	command.RunCobraCLI(false)
}
