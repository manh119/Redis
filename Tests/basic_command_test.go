package Tests

import (
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/manh119/Redis/server"
)

// Khởi tạo client dùng chung cho các test
func SetupClientAndServer() *redis.Client {
	go server.RunIoMultiplexingServerMultipleIOHanlder(&sync.WaitGroup{})
	time.Sleep(200 * time.Millisecond)

	return redis.NewClient(&redis.Options{
		Addr:         "localhost:4000",
		ReadTimeout:  100 * time.Second,
		WriteTimeout: 100 * time.Second,
	})
}

func TestRedisCommands(t *testing.T) {
	rdb := SetupClientAndServer()

	// 1. Test lệnh SET
	t.Run("TestSET", func(t *testing.T) {
		// Redis SET thường trả về "OK" (định dạng RESP: +OK\r\n)
		err := rdb.Set("name", "gemini", 0).Err()
		if err != nil {
			t.Fatalf("Lỗi SET: %v", err)
		}
	})

	// 2. Test lệnh GET
	t.Run("TestGET", func(t *testing.T) {
		val, err := rdb.Get("name").Result()
		if err != nil {
			t.Fatalf("Lỗi GET: %v", err)
		}
		if val != "gemini" {
			t.Errorf("Mong đợi 'gemini', nhưng nhận được: %s", val)
		}
	})

	// 3. Test lệnh TTL (Time To Live)
	t.Run("TestTTL", func(t *testing.T) {
		// SET key với thời gian sống là 10 giây
		expiration := 10 * time.Second
		err := rdb.Set("temp_key", "value", expiration).Err()
		if err != nil {
			t.Fatalf("Lỗi SET với TTL: %v", err)
		}

		// Kiểm tra TTL
		ttl, err := rdb.TTL("temp_key").Result()
		if err != nil {
			t.Fatalf("Lỗi gọi lệnh TTL: %v", err)
		}

		// TTL trả về kiểu time.Duration.
		// Nó sẽ xấp xỉ 10s (có thể là 9.99s do độ trễ)
		if ttl <= 0 || ttl > expiration {
			t.Errorf("TTL không hợp lệ: %v", ttl)
		}
	})

	// 4. Test GET key không tồn tại (Phải trả về lỗi redis.Nil)
	t.Run("TestGetNotFound", func(t *testing.T) {
		_, err := rdb.Get("non_existent_key").Result()
		if err != redis.Nil {
			t.Errorf("Mong đợi lỗi redis.Nil, nhưng nhận được: %v", err)
		}
	})

	// 1. Test SET & GET cơ bản
	t.Run("BasicSetGet", func(t *testing.T) {
		err := rdb.Set("key1", "value1", 0).Err()
		if err != nil {
			t.Fatalf("Lỗi SET: %v", err)
		}

		val, err := rdb.Get("key1").Result()
		if err != nil {
			t.Fatalf("Lỗi GET: %v", err)
		}
		if val != "value1" {
			t.Errorf("Mong đợi 'value1', nhận được: %s", val)
		}
	})

	// 2. Test ghi đè (Overwrite) giá trị cũ
	t.Run("OverwriteKey", func(t *testing.T) {
		rdb.Set("key2", "old_value", 0)
		err := rdb.Set("key2", "new_value", 0).Err()
		if err != nil {
			t.Fatalf("Lỗi khi ghi đè SET: %v", err)
		}

		val, _ := rdb.Get("key2").Result()
		if val != "new_value" {
			t.Errorf("Giá trị không được cập nhật, nhận được: %s", val)
		}
	})

	// 3. Test với Key không tồn tại
	t.Run("GetNonExistentKey", func(t *testing.T) {
		_, err := rdb.Get("non_existent").Result()
		if err != redis.Nil {
			t.Errorf("Mong đợi lỗi redis.Nil cho key không tồn tại, nhưng nhận được: %v", err)
		}
	})

	// 4. Test với giá trị rỗng hoặc Key có khoảng trắng
	t.Run("SpecialValues", func(t *testing.T) {
		err := rdb.Set("empty_val", "", 0).Err()
		if err != nil {
			t.Fatalf("Lỗi SET giá trị rỗng: %v", err)
		}

		val, _ := rdb.Get("empty_val").Result()
		if val != "" {
			t.Errorf("Mong đợi chuỗi rỗng, nhận được: '%s'", val)
		}
	})

	// 5. Test tính đồng thời (Concurrency) - Tùy chọn
	// Kiểm tra xem IoMultiplexingServer có thực sự xử lý được nhiều request cùng lúc không
	t.Run("ConcurrentSet", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				rdb.Set("concurrent_key", id, 0)
				done <- true
			}(i)
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
