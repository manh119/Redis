package tests

import (
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/manh119/Redis/server"
)

// Khởi tạo client dùng chung cho các test
func SetupClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:4000",
	})
}

func TestRedisCommands(t *testing.T) {
	// Khởi động server (chỉ chạy một lần nếu các test nằm chung)
	go server.RunIoMultiplexingServer()
	time.Sleep(200 * time.Millisecond)

	rdb := SetupClient()

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
}

func TestRedisCommands22(t *testing.T) {
	// Khởi động server
	go server.RunIoMultiplexingServer()
	time.Sleep(200 * time.Millisecond)

	rdb := SetupClient()

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

func TestRedisAdvancedCommands(t *testing.T) {
	rdb := SetupClient()
	//ctx := context.Background() // v6 có thể không cần ctx nhưng nên tập thói quen dùng

	// --- NHÓM TEST EXISTS (5 cases) ---
	t.Run("EXISTS", func(t *testing.T) {
		// 1. Key không tồn tại
		if count, _ := rdb.Exists("non_existent").Result(); count != 0 {
			t.Errorf("Case 1: Mong đợi 0, nhận %d", count)
		}
		// 2. Key có tồn tại
		rdb.Set("key1", "val", 0)
		if count, _ := rdb.Exists("key1").Result(); count != 1 {
			t.Errorf("Case 2: Mong đợi 1, nhận %d", count)
		}
		// 3. Kiểm tra nhiều key (trả về số lượng key tồn tại)
		rdb.Set("key2", "val", 0)
		if count, _ := rdb.Exists("key1", "key2", "key3").Result(); count != 2 {
			t.Errorf("Case 3: Mong đợi 2, nhận %d", count)
		}
		// 4. Kiểm tra key sau khi bị ghi đè
		rdb.Set("key1", "new_val", 0)
		if count, _ := rdb.Exists("key1").Result(); count != 1 {
			t.Errorf("Case 4: Vẫn phải là 1")
		}
		// 5. Kiểm tra key với tên là chuỗi rỗng
		rdb.Set("", "empty", 0)
		if count, _ := rdb.Exists("").Result(); count != 1 {
			t.Errorf("Case 5: Key rỗng vẫn phải tồn tại")
		}
	})

	// --- NHÓM TEST DEL (5 cases) ---
	t.Run("DEL", func(t *testing.T) {
		// 6. Xóa một key đang tồn tại
		rdb.Set("del1", "v", 0)
		if n, _ := rdb.Del("del1").Result(); n != 1 {
			t.Errorf("Case 6: Xóa 1 key thành công phải trả về 1")
		}
		// 7. Xóa key không tồn tại
		if n, _ := rdb.Del("ghost").Result(); n != 0 {
			t.Errorf("Case 7: Xóa key không tồn tại phải trả về 0")
		}
		// 8. Xóa nhiều key cùng lúc
		rdb.Set("a", "1", 0)
		rdb.Set("b", "2", 0)
		rdb.Set("c", "3", 0)
		if n, _ := rdb.Del("a", "b", "d").Result(); n != 2 {
			t.Errorf("Case 8: Chỉ xóa được 2 key tồn tại, nhận %d", n)
		}
		// 9. Kiểm tra GET sau khi DEL
		rdb.Set("temp", "val", 0)
		rdb.Del("temp")
		if _, err := rdb.Get("temp").Result(); err != redis.Nil {
			t.Errorf("Case 9: Sau khi xóa GET phải trả về redis.Nil")
		}
		// 10. Xóa cùng một key nhiều lần trong 1 lệnh
		rdb.Set("dup", "v", 0)
		if n, _ := rdb.Del("dup", "dup").Result(); n != 1 {
			t.Errorf("Case 10: Xóa trùng lặp chỉ tính 1 lần thành công")
		}
	})

	// --- NHÓM TEST EXPIRE (10 cases) ---
	t.Run("EXPIRE", func(t *testing.T) {
		// 11. Đặt Expire cho key tồn tại
		rdb.Set("exp1", "v", 0)
		if ok, _ := rdb.Expire("exp1", 10*time.Second).Result(); !ok {
			t.Error("Case 11: Expire phải thành công (true)")
		}
		// 12. Đặt Expire cho key KHÔNG tồn tại
		if ok, _ := rdb.Expire("ghost", 10*time.Second).Result(); ok {
			t.Error("Case 12: Expire key không tồn tại phải trả về false")
		}
		// 13. Kiểm tra key thực sự biến mất sau khi hết hạn
		rdb.Set("short_lived", "v", 100*time.Millisecond)
		time.Sleep(200 * time.Millisecond)
		if rdb.Exists("short_lived").Val() != 0 {
			t.Error("Case 13: Key phải biến mất sau Sleep")
		}
		// 14. EXPIRE với giá trị 0 giây (Xóa key ngay lập tức)
		rdb.Set("exp0", "v", 0)
		rdb.Expire("exp0", 0)
		if rdb.Exists("exp0").Val() != 0 {
			t.Error("Case 14: Expire 0 phải xóa key")
		}
		// 15. EXPIRE với giá trị âm (Cũng xóa key ngay lập tức)
		rdb.Set("exp_neg", "v", 0)
		rdb.Expire("exp_neg", -5*time.Second)
		if rdb.Exists("exp_neg").Val() != 0 {
			t.Error("Case 15: Expire âm phải xóa key")
		}
		// 16. Cập nhật lại thời gian EXPIRE (Overwrite TTL)
		rdb.Set("update_exp", "v", 100*time.Second)
		rdb.Expire("update_exp", 5*time.Second)
		if ttl := rdb.TTL("update_exp").Val(); ttl > 6*time.Second {
			t.Error("Case 16: TTL không được cập nhật mới")
		}
		// 17. Lệnh PERSIST (Gỡ bỏ Expire)
		rdb.Set("per", "v", 10*time.Second)
		rdb.Persist("per")
		if ttl := rdb.TTL("per").Val(); ttl != -1 {
			t.Errorf("Case 17: Key Persist phải có TTL = -1, nhận %v", ttl)
		}
		// 18. SET lại key có đang có Expire (Mất Expire cũ)
		rdb.Set("reset", "v", 10*time.Second)
		rdb.Set("reset", "new_v", 0) // SET mặc định xóa TTL
		if ttl := rdb.TTL("reset").Val(); ttl != -1 {
			t.Error("Case 18: Lệnh SET phải làm mất TTL cũ")
		}
		// 19. Kiểm tra TTL của key không tồn tại
		if ttl := rdb.TTL("no_key").Val(); ttl != -2 {
			t.Errorf("Case 19: TTL key không tồn tại phải là -2, nhận %v", ttl)
		}
		// 20. Expire một key đã có Expire ngắn hơn (Gia hạn)
		rdb.Set("extend", "v", 1*time.Second)
		rdb.Expire("extend", 100*time.Second)
		if rdb.TTL("extend").Val() < 10*time.Second {
			t.Error("Case 20: TTL không được gia hạn")
		}
	})
}
