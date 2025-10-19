package cronjob

import (
	"context"
	"fmt"
	"time"
)

// taskBackupDatabase mô phỏng việc backup database
func taskBackupDatabase() {
	fmt.Println("   💾 Đang backup database...")
	time.Sleep(1 * time.Second) // Giả lập thời gian xử lý
	fmt.Println("   ✅ Backup database hoàn thành!")
}

// taskSendEmail mô phỏng việc gửi email
func taskSendEmail() {
	fmt.Println("   📧 Đang gửi email báo cáo...")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("   ✅ Đã gửi email thành công!")
}

// jobBackupDatabase thực hiện backup database
func jobBackupDatabase() {
	fmt.Println("\n💾 [BACKUP] Bắt đầu backup database...")

	// Giả lập quá trình backup
	databases := []string{"users", "products", "orders", "analytics"}

	for i, db := range databases {
		fmt.Printf("   [%d/%d] Đang backup bảng: %s\n", i+1, len(databases), db)
		time.Sleep(300 * time.Millisecond)
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("   ✅ Backup hoàn thành lúc %s\n", timestamp)
}

// jobSendDailyReport gửi báo cáo hàng ngày
func jobSendDailyReport() {
	fmt.Println("\n📊 [REPORT] Đang tạo và gửi báo cáo hàng ngày...")

	reports := map[string]interface{}{
		"Tổng đơn hàng":    1250,
		"Doanh thu (VND)":  125000000,
		"Người dùng mới":   87,
		"Tỷ lệ chuyển đổi": "3.2%",
	}

	fmt.Println("   📈 Nội dung báo cáo:")
	for key, value := range reports {
		fmt.Printf("      • %s: %v\n", key, value)
	}

	time.Sleep(500 * time.Millisecond)
	fmt.Println("   📧 Đã gửi email đến: admin@company.com")
	fmt.Println("   ✅ Hoàn thành!")
}

// DISTRIBUTED LOCKING EXAMPLE

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
