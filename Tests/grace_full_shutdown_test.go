package Tests

import (
	"log"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/manh119/Redis/server"
)

func SetupServerForTesting() chan os.Signal {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	// Tạo channel để giả lập tín hiệu hệ điều hành
	sig := make(chan os.Signal, 1)

	// Chạy server trong một goroutine
	go func() {
		// Lưu ý: Trong thực tế, bạn nên truyền wg và port vào RunIoMultiplexingServer
		server.RunIoMultiplexingServer(wg)
	}()

	// Goroutine lắng nghe tín hiệu để thực hiện shutdown logic
	go func() {
		<-sig
		log.Println("Test shutdown signal received")
		server.ShutdownAsap.Store(1)
		// Có thể cần đóng listener ở đây nếu server.Run không tự thoát
		wg.Wait()
		log.Println("Grace shutdown completed")
	}()

	// Đợi một chút để server kịp Bind port trước khi trả về cho Test
	time.Sleep(50 * time.Millisecond)

	return sig
}

func TestGracefulShutdown(t *testing.T) {
	// Giả định SetupServer trả về instance server và một channel để gửi tín hiệu dừng
	// srv, shutdownSig := SetupServerForTesting()

	// 1. Test đóng listener mới
	t.Run("TestStopAcceptingNewConnections", func(t *testing.T) {
		shutdownSig := SetupServerForTesting()
		shutdownSig <- syscall.SIGTERM // Kích hoạt shutdown

		time.Sleep(100 * time.Millisecond) // Đợi server chuyển trạng thái

		rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
		time.Sleep(1009 * time.Millisecond)
		result, _ := rdb.Ping().Result()

		if result == "PONG" {
			t.Fatal("Server vẫn chấp nhận kết nối mới sau khi shutdown")
		}

	})

	//// 2. Test hoàn thành request đang xử lý (Long-running)
	//t.Run("TestCompleteInflightRequests", func(t *testing.T) {
	//	srv, shutdownSig := SetupServerForTesting()
	//	rdb := SetupClient()
	//
	//	// Gửi một lệnh giả lập xử lý lâu (nếu server bạn hỗ trợ lệnh như DEBUG SLEEP)
	//	go func() {
	//		rdb.Do("DEBUG", "SLEEP", "0.5").Result()
	//	}()
	//
	//	time.Sleep(100 * time.Millisecond)
	//	shutdownSig <- syscall.SIGTERM // Shutdown khi lệnh đang chạy
	//
	//	val, err := rdb.Get("key").Result() // Kiểm tra xem lệnh sau đó có lỗi không hoặc lệnh hiện tại có xong không
	//	// Logic kiểm tra tùy thuộc vào việc server có đợi request cũ hoàn tất hay không
	//})
	//
	//// 3. Test giải phóng tài nguyên (Close DB/File)
	//t.Run("TestResourceCleanup", func(t *testing.T) {
	//	srv, _ := SetupServerForTesting()
	//	srv.Shutdown() // Gọi trực tiếp hàm shutdown
	//
	//	if srv.DB.IsOpen() {
	//		t.Error("Database connection chưa được đóng sau shutdown")
	//	}
	//})
	//
	//// 4. Test Persistence (RDB/AOF) trước khi thoát
	//t.Run("TestSaveDataBeforeExit", func(t *testing.T) {
	//	srv, shutdownSig := SetupServerForTesting()
	//	rdb := SetupClient()
	//	rdb.Set("persist_key", "important_data", 0)
	//
	//	shutdownSig <- syscall.SIGTERM
	//	time.Sleep(500 * time.Millisecond) // Đợi flush dữ liệu
	//
	//	// Kiểm tra file dump có tồn tại và chứa dữ liệu không
	//	if !FileExists("dump.rdb") {
	//		t.Error("Server thoát mà chưa kịp lưu dữ liệu (RDB)")
	//	}
	//})
	//
	//// 5. Test ngắt kết nối Pub/Sub
	//t.Run("TestPubSubTermination", func(t *testing.T) {
	//	srv, shutdownSig := SetupServerForTesting()
	//	pubsub := SetupClient().Subscribe("channel1")
	//
	//	shutdownSig <- syscall.SIGTERM
	//
	//	_, err := pubsub.ReceiveTimeout(1 * time.Second)
	//	if err == nil {
	//		t.Error("Kết nối Pub/Sub vẫn treo sau khi server shutdown")
	//	}
	//})
	//
	//// 6. Test Timeout Shutdown (Force Kill)
	//t.Run("TestShutdownTimeout", func(t *testing.T) {
	//	// Test trường hợp các tiến trình con bị kẹt, server phải force exit sau X giây
	//	srv, _ := SetupServerWithTimeout(2 * time.Second)
	//	start := time.Now()
	//	srv.Shutdown() // Giả lập một task bị kẹt
	//
	//	duration := time.Since(start)
	//	if duration > 3*time.Second {
	//		t.Errorf("Shutdown mất quá nhiều thời gian: %v", duration)
	//	}
	//})
	//
	//// 7. Test xử lý nhiều tín hiệu dồn dập
	//t.Run("TestMultipleSignals", func(t *testing.T) {
	//	_, shutdownSig := SetupServerForTesting()
	//	// Gửi liên tiếp nhiều tín hiệu
	//	shutdownSig <- syscall.SIGTERM
	//	shutdownSig <- syscall.SIGINT
	//	shutdownSig <- syscall.SIGTERM
	//
	//	// Kiểm tra xem server có bị panic do đóng channel 2 lần không
	//})
	//
	//// 8. Test Client nhận phản hồi "Server is shutting down"
	//t.Run("TestClientErrorMessageOnShutdown", func(t *testing.T) {
	//	srv, shutdownSig := SetupServerForTesting()
	//	rdb := SetupClient()
	//
	//	shutdownSig <- syscall.SIGTERM
	//	_, err := rdb.Set("test", "val", 0).Result()
	//
	//	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
	//		t.Log("Nhận lỗi mong đợi khi kết nối bị đóng")
	//	}
	//})
	//
	//// 9. Test trạng thái Read-only trong quá trình shutdown
	//t.Run("TestReadOnlyStateDuringShutdown", func(t *testing.T) {
	//	// Nếu server có trạng thái trung gian: dừng ghi nhưng vẫn cho phép đọc nốt
	//	// Kiểm tra logic nội bộ của server tại đây
	//})
	//
	//// 10. Test giải phóng Port (Zombie process check)
	//t.Run("TestPortRelease", func(t *testing.T) {
	//	srv, _ := SetupServerForTesting()
	//	srv.Shutdown()
	//
	//	// Thử bind lại port đó ngay lập tức
	//	ln, err := net.Listen("tcp", ":6379")
	//	if err != nil {
	//		t.Errorf("Port chưa được giải phóng: %v", err)
	//	}
	//	if ln != nil {
	//		ln.Close()
	//	}
	//})
}
