package Tests

import (
	"net"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/manh119/Redis/internal/config"
	"github.com/manh119/Redis/server"
)

func TestMain(m *testing.M) {
	// disable parallel tests
	os.Exit(m.Run())
}

func SetupServerForTesting() chan os.Signal {
	wg := &sync.WaitGroup{}
	sig := make(chan os.Signal, 1)
	wg.Add(2)

	go func() {
		server.RunIoMultiplexingServer(wg)
	}()

	go func() {
		server.HandleShutDown(sig, wg)
	}()
	server.ServerStatus = config.ServerStatusIdle
	time.Sleep(100 * time.Millisecond)
	return sig
}

func TestGracefulShutdown(t *testing.T) {
	// Giả định SetupServer trả về instance server và một channel để gửi tín hiệu dừng
	// srv, shutdownSig := SetupServerForTesting()

	// 1. Test đóng listener mới
	t.Run("TestStopAcceptingNewConnections", func(t *testing.T) {
		sig := SetupServerForTesting()
		sig <- syscall.SIGTERM

		// Chờ server xác nhận đã đóng listener (ví dụ qua channel hoặc WaitGroup)
		time.Sleep(500 * time.Millisecond)

		rdb := redis.NewClient(&redis.Options{Addr: "localhost:4000"})
		result, err := rdb.Ping().Result()

		if err == nil && result == "PONG" {
			t.Fatal("Server vẫn chấp nhận kết nối mới sau khi shutdown")
		}
	})

	// 2. Test hoàn thành request đang xử lý (Long-running)
	t.Run("TestCompleteInflightRequests", func(t *testing.T) {
		sig := SetupServerForTesting()
		rdb := redis.NewClient(&redis.Options{Addr: "localhost:4000"})

		done := make(chan string, 1)

		go func() {
			res, err := rdb.Ping().Result()
			if err != nil {
				t.Logf("Ping error: %v", err)
			}
			done <- res
		}()

		time.Sleep(300 * time.Millisecond)
		sig <- syscall.SIGTERM

		// chờ kết quả từ goroutine thay vì sleep cứng
		select {
		case result := <-done:
			if result != "PONG" {
				t.Fatal("Server didn't complete on-going command")
			}
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for Ping result")
		}
		time.Sleep(1 * time.Second)
	})

	// 10. Test giải phóng Port (Zombie process check)
	t.Run("TestPortRelease", func(t *testing.T) {
		sig := SetupServerForTesting()
		redis.NewClient(&redis.Options{Addr: "localhost:4000"})

		sig <- syscall.SIGTERM
		time.Sleep(200 * time.Millisecond)

		ln, err := net.Listen("tcp", ":4000")
		if err != nil {
			t.Errorf("Port chưa được giải phóng: %v", err)
		}
		if ln != nil {
			ln.Close()
		}
	})

}
