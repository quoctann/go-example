package main

import (
	"go-example/cronjob"
	"go-example/redis"
)

func main() {
	cronjob.RunNotUseLibCronJobExample(true)
	cronjob.RunUseLibCronJobExample(false)

	redis.RunRedisExample(true)
}
