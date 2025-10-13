package cronjob

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/redis/go-redis/v9"
)

/*
TÌNH HUỐNG:
- Bạn có 3 servers chạy cùng 1 cronjob
- Nếu không có lock: job chạy 3 lần trên 3 servers (BAD!)
- Với Redis lock: chỉ 1 server chạy, 2 server chờ (GOOD!)

CÁCH HOẠT ĐỘNG:
1. Server A cố lock Redis key "job:backup:lock"
2. Nếu lock thành công → Server A chạy job
3. Server B, C thử lock nhưng thất bại → chúng chờ
4. Server A xong → unlock
5. Lần sau, server B có thể lock và chạy
*/

// ========== REDIS LOCK MANAGER ==========

// RedisLockManager quản lý distributed lock bằng Redis
type RedisLockManager struct {
	client       *redis.Client
	instanceID   string        // ID duy nhất của instance này
	lockPrefix   string        // Prefix cho lock key
	lockTTL      time.Duration // Time-to-live cho lock (an toàn nếu crash)
	lockWaitTime time.Duration // Thời gian chờ để acquire lock
}

// NewRedisLockManager tạo lock manager mới
func NewRedisLockManager(redisAddr, instanceID string) (*RedisLockManager, error) {
	// Kết nối đến Redis
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       0,
	})

	// Test kết nối
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("không thể kết nối Redis: %v", err)
	}

	fmt.Printf("✅ Kết nối Redis thành công! (Addr: %s)\n", redisAddr)

	return &RedisLockManager{
		client:       client,
		instanceID:   instanceID,
		lockPrefix:   "cronjob:lock:",
		lockTTL:      2 * time.Minute,  // Lock tồn tại 2 phút (nếu crash auto release)
		lockWaitTime: 30 * time.Second, // Chờ tối đa 30 giây
	}, nil
}

// AcquireLock cố gắng acquire lock cho job
// Trả về true nếu lock thành công, false nếu thất bại
func (r *RedisLockManager) AcquireLock(ctx context.Context, jobName string) (bool, error) {
	lockKey := r.lockPrefix + jobName
	lockValue := r.instanceID + ":" + time.Now().Format("2006-01-02 15:04:05")

	// Thử lock với SET NX (chỉ set nếu key chưa tồn tại)
	result, err := r.client.SetNX(ctx, lockKey, lockValue, r.lockTTL).Result()
	if err != nil {
		return false, err
	}

	return result, nil
}

// ReleaseLock giải phóng lock
func (r *RedisLockManager) ReleaseLock(ctx context.Context, jobName string) error {
	lockKey := r.lockPrefix + jobName

	// Xóa lock key
	err := r.client.Del(ctx, lockKey).Err()
	if err != nil {
		return err
	}

	return nil
}

// TryAcquireLockWithWait cố gắng acquire lock, có chờ đợi
// Trả về true nếu thành công trong thời gian chờ
func (r *RedisLockManager) TryAcquireLockWithWait(ctx context.Context, jobName string) bool {
	deadline := time.Now().Add(r.lockWaitTime)

	for {
		// Thử acquire lock
		acquired, err := r.AcquireLock(ctx, jobName)
		if err == nil && acquired {
			return true
		}

		// Nếu quá thời gian chờ, return false
		if time.Now().After(deadline) {
			return false
		}

		// Chờ 1 giây rồi thử lại
		time.Sleep(1 * time.Second)
	}
}

// GetLockInfo lấy thông tin lock hiện tại
func (r *RedisLockManager) GetLockInfo(ctx context.Context, jobName string) (string, error) {
	lockKey := r.lockPrefix + jobName

	val, err := r.client.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		return "", nil // Lock không tồn tại
	}

	return val, err
}

// Close đóng kết nối Redis
func (r *RedisLockManager) Close() error {
	return r.client.Close()
}

// ========== CÁC HÀM JOB PHÂN TÁN ==========

// BackupDatabase job backup (chỉ 1 instance chạy)
func BackupDatabase(lockMgr *RedisLockManager) {
	ctx := context.Background()
	jobName := "backup-database"

	fmt.Printf("\n⏳ [BACKUP] Instance %s cố lấy lock...\n", lockMgr.instanceID)

	// Cố acquire lock
	acquired, err := lockMgr.AcquireLock(ctx, jobName)
	if err != nil {
		fmt.Printf("❌ [BACKUP] Lỗi acquire lock: %v\n", err)
		return
	}

	if !acquired {
		fmt.Printf("⏭️  [BACKUP] Instance %s không lấy được lock (instance khác đang chạy)\n",
			lockMgr.instanceID)
		return
	}

	fmt.Printf("🔐 [BACKUP] Instance %s đã lấy lock! Bắt đầu backup...\n", lockMgr.instanceID)

	defer func() {
		// Đảm bảo unlock khi xong
		lockMgr.ReleaseLock(ctx, jobName)
		fmt.Printf("🔓 [BACKUP] Instance %s đã unlock\n", lockMgr.instanceID)
	}()

	// Simulate backup
	databases := []string{"users", "products", "orders"}
	for i, db := range databases {
		fmt.Printf("   [%d/%d] Backing up: %s\n", i+1, len(databases), db)
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("   ✅ Backup hoàn thành lúc %s\n", time.Now().Format("15:04:05"))
}

// SyncExternalData job sync dữ liệu từ bên ngoài (only 1 instance)
func SyncExternalData(lockMgr *RedisLockManager) {
	ctx := context.Background()
	jobName := "sync-external-data"

	fmt.Printf("\n⏳ [SYNC] Instance %s cố lấy lock...\n", lockMgr.instanceID)

	// Lần này dùng TryAcquireLockWithWait - chờ nếu lock bị occupied
	acquired := lockMgr.TryAcquireLockWithWait(ctx, jobName)

	if !acquired {
		fmt.Printf("⏭️  [SYNC] Instance %s không lấy được lock sau %v giây\n",
			lockMgr.instanceID, lockMgr.lockWaitTime)
		return
	}

	fmt.Printf("🔐 [SYNC] Instance %s đã lấy lock! Bắt đầu sync...\n", lockMgr.instanceID)

	defer func() {
		lockMgr.ReleaseLock(ctx, jobName)
		fmt.Printf("🔓 [SYNC] Instance %s đã unlock\n", lockMgr.instanceID)
	}()

	// Simulate sync
	endpoints := []string{"API 1", "API 2", "API 3"}
	for i, endpoint := range endpoints {
		fmt.Printf("   [%d/%d] Syncing from: %s\n", i+1, len(endpoints), endpoint)
		time.Sleep(800 * time.Millisecond)
	}

	fmt.Printf("   ✅ Sync hoàn thành lúc %s\n", time.Now().Format("15:04:05"))
}

// GenerateReport job generate report (có thể chạy trên nhiều instance)
func GenerateReport(instanceID string) {
	fmt.Printf("\n📊 [REPORT] Instance %s đang generate report...\n", instanceID)

	time.Sleep(1 * time.Second)

	fmt.Printf("   ✅ Report generated lúc %s\n", time.Now().Format("15:04:05"))
}

// HealthCheck job kiểm tra sức khỏe (chạy trên tất cả instance)
func HealthCheck(instanceID string) {
	fmt.Printf("\n🏥 [HEALTH] Instance %s: System OK\n", instanceID)
}

// ========== UTILITY FUNCTIONS ==========

// MonitorLocks hiển thị trạng thái của tất cả locks
func MonitorLocks(lockMgr *RedisLockManager) {
	ctx := context.Background()

	jobs := []string{"backup-database", "sync-external-data"}

	fmt.Println("\n📋 [MONITOR] Trạng thái locks:")
	for _, job := range jobs {
		info, _ := lockMgr.GetLockInfo(ctx, job)
		if info == "" {
			fmt.Printf("   • %s: 🔓 UNLOCKED\n", job)
		} else {
			fmt.Printf("   • %s: 🔐 LOCKED by %s\n", job, info)
		}
	}
}

// ========== MAIN FUNCTION ==========

// Distributed Cronjob with Redis Lock
func RunCronRedisLock(isSkip bool) {
	if isSkip {
		return
	}

	fmt.Println("🚀 Distributed Cronjob with Redis Lock")
	fmt.Println("=" + string(make([]byte, 70)) + "=\n")

	// ===== CONFIGURATION =====

	// Lấy instance ID từ environment hoặc command line
	// Khi chạy multiple instances:
	// Terminal 1: INSTANCE_ID=server-1 go run main.go
	// Terminal 2: INSTANCE_ID=server-2 go run main.go
	// Terminal 3: INSTANCE_ID=server-3 go run main.go
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "server-1" // Mặc định
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Mặc định
	}

	fmt.Printf("📌 Instance ID: %s\n", instanceID)
	fmt.Printf("📌 Redis Address: %s\n\n", redisAddr)

	// ===== KHỞI TẠO REDIS LOCK MANAGER =====

	lockMgr, err := NewRedisLockManager(redisAddr, instanceID)
	if err != nil {
		log.Fatal("Lỗi khởi tạo Redis:", err)
	}
	defer lockMgr.Close()

	// ===== KHỞI TẠO SCHEDULER =====

	location, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	s, err := gocron.NewScheduler(
		gocron.WithLocation(location),
	)
	if err != nil {
		log.Fatal("Lỗi tạo scheduler:", err)
	}

	// ===== THÊM DISTRIBUTED JOBS =====

	// Job 1: Backup Database (chỉ 1 instance chạy)
	// Wrap function để pass lockMgr
	job1, _ := s.NewJob(
		gocron.DurationJob(15*time.Second),
		gocron.NewTask(func() { BackupDatabase(lockMgr) }),
		gocron.WithName("Backup Database"),
		gocron.WithTags("distributed", "critical"),
	)
	fmt.Printf("✅ Job 1: %s - Mỗi 15 giây (distributed)\n", job1.Name())

	// Job 2: Sync External Data (chỉ 1 instance chạy)
	job2, _ := s.NewJob(
		gocron.DurationJob(20*time.Second),
		gocron.NewTask(func() { SyncExternalData(lockMgr) }),
		gocron.WithName("Sync External Data"),
		gocron.WithTags("distributed", "critical"),
	)
	fmt.Printf("✅ Job 2: %s - Mỗi 20 giây (distributed)\n", job2.Name())

	// Job 3: Generate Report (chạy trên TẤT CẢ instances)
	// Không cần lock
	job3, _ := s.NewJob(
		gocron.DurationJob(25*time.Second),
		gocron.NewTask(func() { GenerateReport(instanceID) }),
		gocron.WithName("Generate Report"),
		gocron.WithTags("local"),
	)
	fmt.Printf("✅ Job 3: %s - Mỗi 25 giây (local, tất cả instances)\n", job3.Name())

	// Job 4: Health Check (chạy trên TẤT CẢ instances)
	job4, _ := s.NewJob(
		gocron.DurationJob(10*time.Second),
		gocron.NewTask(func() { HealthCheck(instanceID) }),
		gocron.WithName("Health Check"),
		gocron.WithTags("local"),
	)
	fmt.Printf("✅ Job 4: %s - Mỗi 10 giây (local, tất cả instances)\n", job4.Name())

	// Job 5: Monitor Locks (chạy trên tất cả instances)
	job5, _ := s.NewJob(
		gocron.DurationJob(30*time.Second),
		gocron.NewTask(func() { MonitorLocks(lockMgr) }),
		gocron.WithName("Monitor Locks"),
		gocron.WithTags("monitoring"),
	)
	fmt.Printf("✅ Job 5: %s - Mỗi 30 giây (monitoring)\n", job5.Name())

	fmt.Println("\n" + string(make([]byte, 70)) + "")
	fmt.Println("\n💡 HƯỚNG DẪN CHẠY MULTIPLE INSTANCES:")
	fmt.Println("   Mở 3 terminal khác nhau và chạy:")
	fmt.Println("   Terminal 1: INSTANCE_ID=server-1 go run main.go")
	fmt.Println("   Terminal 2: INSTANCE_ID=server-2 go run main.go")
	fmt.Println("   Terminal 3: INSTANCE_ID=server-3 go run main.go")
	fmt.Println("\n   Quan sát:")
	fmt.Println("   • Jobs với tag 'distributed': chỉ 1 server chạy")
	fmt.Println("   • Jobs với tag 'local': tất cả servers đều chạy")
	fmt.Println("   • Jobs với tag 'monitoring': xem ai đang giữ lock")
	fmt.Println("\n💡 Nhấn Ctrl+C để dừng")

	// Start scheduler
	s.Start()

	// ===== DEMO: Kiểm tra lock sau vài giây =====

	go func() {
		time.Sleep(5 * time.Second)
		fmt.Println("\n" + string(make([]byte, 70)) + "")
		fmt.Println("📊 [DEMO] Redis Lock Info:")

		ctx := context.Background()
		lockMgr.client.FlushDB(ctx) // Clear dùng cho demo

		info, _ := lockMgr.client.Info(ctx, "server").Result()
		fmt.Printf("   Redis Info: Connected ✅ %v\n", info)

		// Liệt kê tất cả keys
		keys, _ := lockMgr.client.Keys(ctx, "cronjob:lock:*").Result()
		fmt.Printf("   Active locks: %d\n", len(keys))
		for _, key := range keys {
			val, _ := lockMgr.client.Get(ctx, key).Result()
			fmt.Printf("      🔐 %s -> %s\n", key, val)
		}
	}()

	// Chạy trong 1 phút rồi dừng (demo)
	time.Sleep(1 * time.Minute)

	fmt.Println("\n🛑 Shutting down...")
	s.Shutdown()
	fmt.Println("✅ Done!")
}

/*
CHẠY DEMO:
   # Terminal 1
   INSTANCE_ID=server-1 go run main.go

   # Terminal 2 (tại cùng lúc)
   INSTANCE_ID=server-2 go run main.go

   # Terminal 3 (tùy chọn)
   INSTANCE_ID=server-3 go run main.go

QUAN SÁT KẾT QUẢ:
   ✅ "Backup Database" và "Sync External Data":
      - Chỉ 1 server chạy
      - Server khác thấy "không lấy được lock"
      - Lần chạy tiếp, server khác có thể chạy

   ✅ "Generate Report" và "Health Check":
      - Tất cả server đều chạy
      - Không cần lock

   ✅ "Monitor Locks":
      - Hiển thị ai đang giữ lock
      - Useful cho debugging

LOCK MECHANISM:

   Redis SetNX:
   - SET key value NX -> Set nếu key chưa tồn tại
   - Nếu thành công: return 1 (lấy được lock)
   - Nếu thất bại: return 0 (lock bị occupied)

   Example:
   SET cronjob:lock:backup-database "server-1:2025-10-13 14:30:00" NX EX 120

   Meaning:
   - key: cronjob:lock:backup-database
   - value: server-1:2025-10-13 14:30:00
   - NX: chỉ set nếu key chưa tồn tại
   - EX 120: expire sau 120 giây (auto release nếu crash)

PRODUCTION CONSIDERATIONS:

   ✅ TTL (Time-To-Live):
      - Set TTL vừa đủ lâu để job chạy xong
      - Không quá dài (nếu crash, phải chờ lâu)
      - Hiện tại: 2 phút (có thể adjust)

   ✅ Lock Wait Time:
      - Hiện tại: 30 giây
      - Nếu job thường chạy lâu, tăng lên

   ✅ Error Handling:
      - Nếu Redis down: jobs vẫn chạy (không có lock)
      - Xử lý bằng cách connect failover

   ✅ Monitoring:
      - Log tất cả lock acquire/release
      - Alert nếu lock held quá lâu
      - Metrics: lock wait time, lock contention

ADVANCED PATTERNS:

   a) Fair Distribution:
      - Thay vì first-come-first-served
      - Rotating lock ownership

   b) Priority Jobs:
      - High priority: 10 second wait time
      - Low priority: 60 second wait time

   c) Circuit Breaker:
      - Nếu job fail nhiều lần: skip
      - Prevent cascade failures

   d) Multi-Region:
      - Redis cluster
      - Failover handling
      - Geo-distributed locks

TESTING:

   # Simulate crash (hold lock):
   redis-cli
   SET cronjob:lock:backup-database "test" EX 120

   # Quan sát: server khác chờ hoặc timeout
   # Sau 120 giây: lock auto release

   # Manual unlock:
   DEL cronjob:lock:backup-database

PERFORMANCE TIPS:

   ✅ Use Redis Cluster cho high availability
   ✅ Monitor Redis memory usage
   ✅ Implement job timeout (nếu job hung)
   ✅ Log detailed metrics
   ✅ Alert on lock timeout
*/
