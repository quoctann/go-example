package cronjob

import (
	"fmt"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

// TODO: thêm ví dụ cronjob với redis lock

// MARK: NOT USE LIB

// Task định nghĩa cấu trúc của một công việc cần chạy định kỳ
type Task struct {
	Name     string        // Tên của task
	Interval time.Duration // Khoảng thời gian giữa các lần chạy
	Job      func()        // Hàm sẽ được thực thi
}

// Scheduler quản lý tất cả các task
type Scheduler struct {
	tasks []Task        // Danh sách các task cần chạy
	stop  chan struct{} // Channel để signal dừng tất cả task
}

// NewScheduler tạo một scheduler mới
func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make([]Task, 0),
		stop:  make(chan struct{}), // Khởi tạo stop channel
	}
}

// AddTask thêm một task mới vào scheduler
func (s *Scheduler) AddTask(name string, interval time.Duration, job func()) {
	task := Task{
		Name:     name,
		Interval: interval,
		Job:      job,
	}
	s.tasks = append(s.tasks, task)
	fmt.Printf("✅ Đã thêm task: %s (chạy mỗi %v)\n", name, interval)
}

// Stop dừng tất cả các task một cách graceful
func (s *Scheduler) Stop() {
	fmt.Println("\n🛑 Đang dừng scheduler...")
	close(s.stop) // Đóng channel để signal tất cả goroutines dừng lại
}

// Start bắt đầu chạy tất cả các task
func (s *Scheduler) Start() {
	fmt.Println("🚀 Scheduler đã khởi động!")

	// Với mỗi task, tạo một goroutine riêng để chạy
	for _, task := range s.tasks {
		// Sử dụng goroutine để chạy song song nhiều task
		go s.runTask(task)
	}
}

// runTask chạy một task theo chu kỳ
func (s *Scheduler) runTask(task Task) {
	// Tạo một ticker để kích hoạt sau mỗi khoảng thời gian interval
	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop() // Đảm bảo dọn dẹp ticker khi hàm kết thúc

	// Chạy task ngay lập tức lần đầu tiên (không đợi interval)
	fmt.Printf("⏰ [%s] Chạy lần đầu tiên\n", task.Name)
	task.Job()

	// Vòng lặp để chạy task định kỳ
	for {
		// SỬ DỤNG SELECT để lắng nghe nhiều channel
		// Điều này cho phép graceful shutdown và tránh block vĩnh viễn
		select {
		case <-ticker.C:
			// Khi ticker gửi tín hiệu, chạy task
			fmt.Printf("⏰ [%s] Đang chạy...\n", task.Name)
			task.Job()

		case <-s.stop:
			// Khi nhận signal dừng, thoát goroutine một cách an toàn
			fmt.Printf("🛑 [%s] Đã dừng\n", task.Name)
			return
		}
	}
}

func RunNotUseLibCronJobExample(isSkip bool) {
	if isSkip {
		fmt.Println("\n⚠️ Bỏ qua ví dụ cronjob không dùng lib")
		return
	}
	// Tạo scheduler mới
	scheduler := NewScheduler()

	// Thêm các task với các khoảng thời gian khác nhau
	scheduler.AddTask("Backup Database", 2*time.Second, taskBackupDatabase)
	scheduler.AddTask("Send Email", 4*time.Second, taskSendEmail)

	// Bắt đầu chạy scheduler
	scheduler.Start()

	// Giữ chương trình chạy mãi mãi
	// Trong thực tế, bạn có thể thêm signal handling để dừng gracefully
	fmt.Println("\n💡 Nhấn Ctrl+C để dừng chương trình")

	// TÙY CHỌN 1: Block mãi mãi (đơn giản nhất)
	// select {}

	// TÙY CHỌN 2: Graceful shutdown với timeout (KHUYÊN DÙNG)
	// Chạy trong 30 giây rồi tự động dừng
	time.Sleep(10 * time.Second)
	scheduler.Stop()

	// Đợi một chút để các goroutine cleanup
	time.Sleep(1 * time.Second)
	fmt.Println("✅ Chương trình đã dừng hoàn toàn!")
}

// MARK: USE LIB
func RunUseLibCronJobExample(isSkip bool) {
	if isSkip {
		fmt.Println("\n⚠️ Bỏ qua ví dụ cronjob dùng lib")
		return
	}

	// Tạo scheduler mới với timezone
	// Location giúp đảm bảo job chạy đúng múi giờ
	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		log.Fatal("Lỗi load timezone:", err)
	}

	// Tạo scheduler với options
	s, err := gocron.NewScheduler(
		gocron.WithLocation(location), // Set timezone
	)
	if err != nil {
		log.Fatal("Lỗi tạo scheduler:", err)
	}

	fmt.Println("📍 Timezone:", location)

	// thêm job

	_, err = s.NewJob(
		gocron.DurationJob(3*time.Second),
		gocron.NewTask(jobBackupDatabase),
		gocron.WithName("Database Backup"),
		gocron.WithTags("backup", "critical"),
		gocron.WithSingletonMode(gocron.LimitModeReschedule), // Quan trọng!
		gocron.WithLimitedRuns(3),                            // Chỉ chạy 3 lần rồi tự động dừng
		gocron.WithEventListeners(
			// Thêm event listener để log khi job chạy xong
			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
				fmt.Printf("\n🎯 [EVENT] Job '%s' đã hoàn thành\n", jobName)
			}),
			// Log khi job bị lỗi
			gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
				fmt.Printf("\n❌ [ERROR] Job '%s' gặp lỗi: %v\n", jobName, err)
			}),
		),
	)
	if err != nil {
		log.Fatal("Lỗi tạo Backup job:", err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(1*time.Second),
		gocron.NewTask(jobSendDailyReport),
		gocron.WithName("send daily report"),
		gocron.WithTags("report", "daily"),
		gocron.WithSingletonMode(gocron.LimitModeReschedule), // Quan trọng!
		gocron.WithLimitedRuns(3),                            // Chỉ chạy 3 lần rồi tự động dừng
		gocron.WithEventListeners(
			// Thêm event listener để log khi job chạy xong
			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
				fmt.Printf("\n🎯 [EVENT] Job '%s' đã hoàn thành\n", jobName)
			}),
			// Log khi job bị lỗi
			gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
				fmt.Printf("\n❌ [ERROR] Job '%s' gặp lỗi: %v\n", jobName, err)
			}),
		),
	)
	if err != nil {
		log.Fatal("Lỗi tạo Backup job:", err)
	}

	// Bắt đầu scheduler
	s.Start()

	// Chạy trong 2 phút rồi shutdown (cho demo)
	time.Sleep(10 * time.Second)

	// Graceful shutdown
	fmt.Println("\n🛑 Đang shutdown scheduler...")
	err = s.Shutdown()
	if err != nil {
		log.Fatal("Lỗi shutdown:", err)
	}

	fmt.Println("✅ Scheduler đã dừng hoàn toàn!")
	fmt.Println("👋 Goodbye!")
}
