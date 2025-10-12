package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// TODO: Thêm ví dụ redis replication, clustering

// RedisConfig chứa cấu hình cho Redis connection
type RedisConfig struct {
	Addr         string        // Địa chỉ Redis server (vd: localhost:6379)
	Password     string        // Mật khẩu (để trống nếu không có)
	DB           int           // Database number (0-15)
	PoolSize     int           // Số lượng connection trong pool
	MinIdleConns int           // Số connection tối thiểu luôn giữ sẵn
	MaxRetries   int           // Số lần retry khi lỗi
	DialTimeout  time.Duration // Timeout khi kết nối
	ReadTimeout  time.Duration // Timeout khi đọc
	WriteTimeout time.Duration // Timeout khi ghi
}

// RedisClient wrap redis.Client với các method tiện ích
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient khởi tạo Redis client với cấu hình production
func NewRedisClient(config RedisConfig) (*RedisClient, error) {
	// Tạo Redis client với options
	client := redis.NewClient(&redis.Options{
		Addr:         config.Addr,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,     // Số connection tối đa trong pool
		MinIdleConns: config.MinIdleConns, // Giữ sẵn connection để tránh tạo mới liên tục
		MaxRetries:   config.MaxRetries,   // Tự động retry khi có lỗi network
		DialTimeout:  config.DialTimeout,  // Timeout khi establish connection
		ReadTimeout:  config.ReadTimeout,  // Timeout khi đọc data
		WriteTimeout: config.WriteTimeout, // Timeout khi ghi data

		// Connection pool settings - quan trọng cho production
		PoolTimeout:  30 * time.Second, // Timeout khi chờ connection từ pool
		MaxIdleConns: 10,               // Số connection idle tối đa
	})

	// Test connection ngay khi khởi tạo
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("không thể kết nối Redis: %w", err)
	}

	log.Println("✓ Kết nối Redis thành công")

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close đóng connection pool - quan trọng để cleanup resources
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Set lưu key-value với TTL (Time To Live)
// TTL = 0 nghĩa là không bao giờ expire
func (r *RedisClient) Set(key string, value interface{}, ttl time.Duration) error {
	// context.WithTimeout tạo timeout riêng cho mỗi operation
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel() // Luôn nhớ cancel context để tránh memory leak

	return r.client.Set(ctx, key, value, ttl).Err()
}

// Get lấy giá trị từ key
func (r *RedisClient) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		// redis.Nil nghĩa là key không tồn tại - không phải lỗi nghiêm trọng
		return "", fmt.Errorf("key không tồn tại: %s", key)
	}
	return val, err
}

// SetJSON lưu struct/object dạng JSON - rất hữu ích trong production
func (r *RedisClient) SetJSON(key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	// Marshal object thành JSON string
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("lỗi marshal JSON: %w", err)
	}

	return r.client.Set(ctx, key, jsonData, ttl).Err()
}

// GetJSON lấy và unmarshal JSON về struct
func (r *RedisClient) GetJSON(key string, dest interface{}) error {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("key không tồn tại: %s", key)
	}
	if err != nil {
		return err
	}

	// Unmarshal JSON string về struct
	return json.Unmarshal([]byte(val), dest)
}

// Delete xóa một hoặc nhiều keys
func (r *RedisClient) Delete(keys ...string) error {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	return r.client.Del(ctx, keys...).Err()
}

// Exists kiểm tra key có tồn tại không
func (r *RedisClient) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

// SetNX (Set if Not eXists) - atomic operation, dùng cho distributed lock
func (r *RedisClient) SetNX(key string, value interface{}, ttl time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	// Trả về true nếu set thành công, false nếu key đã tồn tại
	return r.client.SetNX(ctx, key, value, ttl).Result()
}

// Increment tăng giá trị số nguyên - atomic operation, dùng cho counter
func (r *RedisClient) Increment(key string) (int64, error) {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	return r.client.Incr(ctx, key).Result()
}

// SetExpire set thời gian expire cho key đã tồn tại
func (r *RedisClient) SetExpire(key string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	return r.client.Expire(ctx, key, ttl).Err()
}

// GetTTL lấy thời gian còn lại của key (Time To Live)
func (r *RedisClient) GetTTL(key string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	return r.client.TTL(ctx, key).Result()
}

// Pipeline batch nhiều commands thành 1 network roundtrip - tăng performance
func (r *RedisClient) PipelineExample() error {
	ctx, cancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer cancel()

	// Tạo pipeline
	pipe := r.client.Pipeline()

	// Queue các commands (chưa execute)
	pipe.Set(ctx, "key1", "value1", 0)
	pipe.Set(ctx, "key2", "value2", 0)
	pipe.Incr(ctx, "counter")

	// Execute tất cả commands cùng lúc
	_, err := pipe.Exec(ctx)
	return err
}

// HealthCheck kiểm tra Redis health - dùng cho health check endpoint
func (r *RedisClient) HealthCheck() error {
	ctx, cancel := context.WithTimeout(r.ctx, 2*time.Second)
	defer cancel()

	return r.client.Ping(ctx).Err()
}

// GetPoolStats lấy thông tin về connection pool - hữu ích cho monitoring
func (r *RedisClient) GetPoolStats() *redis.PoolStats {
	return r.client.PoolStats()
}

func RunRedisExample(isSkip bool) {
	if isSkip {
		log.Println("\n⚠️ Bỏ qua ví dụ Redis")
		return
	}

	// Cấu hình Redis với giá trị production-ready
	config := RedisConfig{
		Addr:         "localhost:6379",
		Password:     "", // Set password nếu Redis có authentication
		DB:           0,  // Database 0
		PoolSize:     10, // 10 connections cho small app, scale lên cho high traffic
		MinIdleConns: 2,  // Giữ sẵn 2 connections
		MaxRetries:   3,  // Retry 3 lần khi gặp lỗi
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// Khởi tạo Redis client
	redisClient, err := NewRedisClient(config)
	if err != nil {
		log.Fatalf("Lỗi khởi tạo Redis: %v", err)
	}
	defer redisClient.Close() // Đảm bảo đóng connections khi thoát

	runExample(redisClient)
}
