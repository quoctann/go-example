package queue

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"go-example/config"
	"go-example/email"
	"go-example/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

/*
	Mail queue with redis

	LPUSH: push on the end of queue
	RPOP: get the first of queue
	BRPOP: like RPOP but block until has a job exist on queue

	Advanced:
	Redis Streams
	Dead-Letter queue
	Retry + backoff
	Monitoring queue length (for scaling)
	Idempotency: ensure job only processed once (save success job id into db)

	To run this example:
	1.	Toggle skip flag in main.go file
	2.	Run go run main.go
	3.	In another terminal, run: go run main.go worker
	4.	Use curl to send email request replace your-email@example.com
			curl "http://localhost:8080/send-email?to=your-email@example.com"
*/

const EMAIL_QUEUE_NAME string = "email_queue"

type Job struct {
	ID      string `json:"id"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// Implementation of queue using redis
type RedisQueue struct {
	Client *redis.Client
}

func NewRedisQueue(addr, pwd string, db int) *RedisQueue {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       db,
	})
	return &RedisQueue{Client: client}
}

// Push a job into queue
func (q *RedisQueue) Enqueue(ctx context.Context, job Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	// push on the end of queue
	return q.Client.LPush(ctx, EMAIL_QUEUE_NAME, payload).Err()
}

// Pop a job out of queue
func (q *RedisQueue) Dequeue(ctx context.Context) (Job, error) {
	var job Job

	//  block until has a job exist on queue, timeout 1s
	result, err := q.Client.BRPop(ctx, 1*time.Second, EMAIL_QUEUE_NAME).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// empty queue, should not error
			return job, nil
		}
		return job, err
	}

	// result is an array [key, value], we need value
	if len(result) < 2 {
		return job, errors.New("invalid result from BRPOP")
	}

	if err = json.Unmarshal([]byte(result[1]), &job); err != nil {
		return job, err
	}
	return job, nil
}

func RunMailQueueExample(skip bool) {
	if skip {
		return
	}

	appCfg, err := config.LoadLocalConfig()
	if err != nil {
		log.Fatal(err)
	}
	cfg := utils.ToNonPointer(appCfg).Redis

	redisQueue := NewRedisQueue(cfg.Address, cfg.Password, cfg.DB)
	ctx := context.Background()
	if _, err := redisQueue.Client.Ping(ctx).Result(); err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
	}

	// run api server
	if len(os.Args) < 2 {
		// replace your-email@example.com
		// curl "http://localhost:8080/send-email?to=your-email@example.com"
		http.HandleFunc("/send-email", func(w http.ResponseWriter, r *http.Request) {
			to := r.URL.Query().Get("to")
			if to == "" {
				http.Error(w, "Missing 'to' parameter", http.StatusBadRequest)
				return
			}

			job := Job{
				ID:      uuid.NewString(),
				To:      to,
				Subject: "Test queue",
				Body:    "Mail queue with redis",
			}
			if err := redisQueue.Enqueue(ctx, job); err != nil {
				log.Printf("Failed: %v", err)
				http.Error(w, "Failed to queue email", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"message": "Email has been queued and will be sent shortly.",
				"job_id":  job.ID,
			})
		})

		log.Println("Producer server starting on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Could not start producer server: %v", err)
		}
		return
	}

	// worker
	flag.NewFlagSet("worker", flag.ContinueOnError)
	if os.Args[1] != "worker" {
		log.Println("Nothing run")
		return
	}

	mailService, err := email.NewEmailService()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Worker start")
	// channel listening shutdown signal (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-quit:
			log.Println("Shutting down worker...")
			return

		default:
			job, err := redisQueue.Dequeue(ctx)
			if err != nil {
				log.Printf("Error dequeuing job: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			if job.ID == "" {
				// queue is empty
				continue
			}

			// For debugging, pause here to inspect job in redis-cli/redis insight if needed
			// log.Printf("Processing job ID: %s for %s. Pausing for 10 seconds to allow inspection...", job.ID, job.To)
			// time.Sleep(10 * time.Second)

			log.Printf("Processing job ID %v for %v", job.ID, job.To)
			if err := mailService.Send(job.To, job.Subject, job.Body); err != nil {
				/*
					In case of failure, can following these practices:
					(1) Log error and skip, missing email is acceptable
					(2) Push failed job in queue (may cause infinite retry)
					(3) (Good one) Push failed job into "dead-letter queue" for later analysis
				*/
				log.Printf("FAILED to send email for job ID %s: %v", job.ID, err)
			} else {
				log.Printf("SUCCESSFULLY sent email for job ID: %s", job.ID)
			}
		}
	}
}
