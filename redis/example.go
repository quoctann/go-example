package redis

import (
	"fmt"
	"log"
	"time"
)

// User struct ví dụ để demo cache object
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func runExample(redisClient *RedisClient) {

	// ===== VÍ DỤ 1: String operations =====
	fmt.Println("\n=== String Operations ===")

	// Set key với TTL 1 giờ
	if err := redisClient.Set("user:1:name", "Nguyễn Văn A", 1*time.Hour); err != nil {
		log.Printf("Lỗi set key: %v", err)
	}

	// Get key
	if name, err := redisClient.Get("user:1:name"); err != nil {
		log.Printf("Lỗi get key: %v", err)
	} else {
		fmt.Printf("Name: %s\n", name)
	}

	// ===== VÍ DỤ 2: JSON operations =====
	fmt.Println("\n=== JSON Operations ===")

	user := User{
		ID:        1,
		Name:      "Nguyễn Văn B",
		Email:     "nguyenvanb@example.com",
		CreatedAt: time.Now(),
	}

	// Lưu object dạng JSON với TTL 30 phút
	if err := redisClient.SetJSON("user:1:profile", user, 30*time.Minute); err != nil {
		log.Printf("Lỗi set JSON: %v", err)
	}

	// Lấy object từ JSON
	var cachedUser User
	if err := redisClient.GetJSON("user:1:profile", &cachedUser); err != nil {
		log.Printf("Lỗi get JSON: %v", err)
	} else {
		fmt.Printf("Cached User: %+v\n", cachedUser)
	}

	// ===== VÍ DỤ 3: Counter operations =====
	fmt.Println("\n=== Counter Operations ===")

	// Increment counter
	count, err := redisClient.Increment("page:views")
	if err != nil {
		log.Printf("Lỗi increment: %v", err)
	} else {
		fmt.Printf("Page views: %d\n", count)
	}

	// ===== VÍ DỤ 4: Distributed Lock với SetNX =====
	fmt.Println("\n=== Distributed Lock ===")

	lockKey := "lock:order:123"
	locked, err := redisClient.SetNX(lockKey, "process-1", 10*time.Second)
	if err != nil {
		log.Printf("Lỗi acquire lock: %v", err)
	} else if locked {
		fmt.Println("✓ Lock acquired, đang xử lý...")
		// Xử lý business logic ở đây
		time.Sleep(2 * time.Second)
		redisClient.Delete(lockKey) // Release lock
		fmt.Println("✓ Lock released")
	} else {
		fmt.Println("✗ Lock đã được process khác giữ")
	}

	// ===== VÍ DỤ 5: TTL operations =====
	fmt.Println("\n=== TTL Operations ===")

	ttl, err := redisClient.GetTTL("user:1:name")
	if err != nil {
		log.Printf("Lỗi get TTL: %v", err)
	} else {
		fmt.Printf("TTL còn lại: %v\n", ttl)
	}

	// ===== VÍ DỤ 6: Pipeline =====
	fmt.Println("\n=== Pipeline Operations ===")

	if err := redisClient.PipelineExample(); err != nil {
		log.Printf("Lỗi pipeline: %v", err)
	} else {
		fmt.Println("✓ Pipeline executed successfully")
	}

	// ===== VÍ DỤ 7: Health Check =====
	fmt.Println("\n=== Health Check ===")

	if err := redisClient.HealthCheck(); err != nil {
		log.Printf("✗ Redis unhealthy: %v", err)
	} else {
		fmt.Println("✓ Redis is healthy")
	}

	// ===== VÍ DỤ 8: Pool Stats =====
	fmt.Println("\n=== Connection Pool Stats ===")

	stats := redisClient.GetPoolStats()
	fmt.Printf("Hits: %d, Misses: %d, Timeouts: %d\n",
		stats.Hits, stats.Misses, stats.Timeouts)
	fmt.Printf("Total Conns: %d, Idle Conns: %d\n",
		stats.TotalConns, stats.IdleConns)

	fmt.Println("\n✓ Demo hoàn tất!")
}
