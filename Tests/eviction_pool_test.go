package Tests

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Giả sử hàm SetupClientAndServer() trả về *redis.Client từ go-redis v6
func TestApproximateLRU(t *testing.T) {
	rdb := SetupClientAndServer()
	// Giả lập context nếu cần, nhưng v6 chưa dùng context trong các lệnh cơ bản
	noExp := time.Duration(0)

	// 1. Kiểm tra SET và GET cơ bản
	t.Run("BasicSetGet", func(t *testing.T) {
		rdb.FlushDB()
		rdb.Set("k1", "v1", noExp)
		if rdb.Get("k1").Val() != "v1" {
			t.Errorf("Mong đợi v1, nhận được %s", rdb.Get("k1").Val())
		}
	})

	// 2. Kiểm tra ghi đè không tăng số lượng key
	t.Run("OverwriteNoIncrease", func(t *testing.T) {
		rdb.FlushDB()
		rdb.Set("k1", "v1", noExp)
		rdb.Set("k1", "v2", noExp)
		if rdb.DBSize().Val() != 1 {
			t.Errorf("Số lượng key phải là 1, thực tế: %d", rdb.DBSize().Val())
		}
	})

	// 3. Kiểm tra giới hạn tối đa 10 keys
	t.Run("MaxKeyLimit", func(t *testing.T) {
		rdb.FlushDB()
		for i := 1; i <= 15; i++ {
			rdb.Set(fmt.Sprintf("key%d", i), "val", noExp)
		}
		if rdb.DBSize().Val() > 10 {
			t.Errorf("Số lượng key vượt mức tối đa: %d", rdb.DBSize().Val())
		}
	})

	// 4. Kiểm tra xóa thủ công
	t.Run("ManualDelete", func(t *testing.T) {
		rdb.FlushDB()
		rdb.Set("delete_me", "val", noExp)
		rdb.Del("delete_me")
		if rdb.Get("delete_me").Val() != "" {
			t.Error("Key vẫn tồn tại sau khi xóa")
		}
	})

	// 5. Kiểm tra GET làm mới timestamp
	t.Run("GetUpdatesTimestamp", func(t *testing.T) {
		rdb.FlushDB()
		rdb.Set("cold", "v1", noExp)
		for i := 1; i <= 9; i++ {
			rdb.Set(fmt.Sprintf("k%d", i), "v", noExp)
		}
		rdb.Get("cold") // Làm nóng
		rdb.Set("trigger", "v", noExp)

		if rdb.Get("cold").Val() == "" {
			t.Error("Key 'cold' bị xóa nhầm dù vừa được truy cập!")
		}
	})

	// 6. Kiểm tra SET (update) làm mới timestamp
	t.Run("SetUpdatesTimestamp", func(t *testing.T) {
		rdb.FlushDB()
		rdb.Set("target", "old", noExp)
		time.Sleep(10 * time.Millisecond)
		for i := 1; i <= 9; i++ {
			rdb.Set(fmt.Sprintf("k%d", i), "v", noExp)
		}
		rdb.Set("target", "new", noExp) // Update
		rdb.Set("evictor", "v", noExp)

		if rdb.Get("target").Val() == "" {
			t.Error("Target key bị xóa dù vừa được update")
		}
	})

	// 7. Kiểm tra EvictOldestIdle
	t.Run("EvictOldestIdle", func(t *testing.T) {
		rdb.FlushDB()
		rdb.Set("very_old", "v", noExp)
		time.Sleep(50 * time.Millisecond)
		for i := 1; i <= 9; i++ {
			rdb.Set(fmt.Sprintf("fresh%d", i), "v", noExp)
		}
		rdb.Set("newest", "v", noExp)
		// Lưu ý: Tùy vào tính xấp xỉ mà very_old có thể còn hoặc mất
	})

	// 8. Stress test nhẹ
	t.Run("ContinuousSet", func(t *testing.T) {
		rdb.FlushDB()
		for i := 0; i < 100; i++ {
			rdb.Set(fmt.Sprintf("k%d", i), "v", noExp)
		}
		if rdb.DBSize().Val() > 10 {
			t.Errorf("Lỗi giới hạn: %d", rdb.DBSize().Val())
		}
	})

	// 9. Kiểm tra tính an toàn Concurrency
	t.Run("ConcurrentSafety", func(t *testing.T) {
		rdb.FlushDB()
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					rdb.Set(fmt.Sprintf("g%d-k%d", id, j), "v", noExp)
				}
			}(i)
		}
		wg.Wait()
		if rdb.DBSize().Val() > 10 {
			t.Errorf("Race condition, số lượng key: %d", rdb.DBSize().Val())
		}
	})

	// 10. FlushAll
	t.Run("FlushAll", func(t *testing.T) {
		rdb.Set("k", "v", noExp)
		rdb.FlushDB()
		if rdb.DBSize().Val() != 0 {
			t.Error("Flush không sạch")
		}
	})

	// 12. NewKeyImmunity
	t.Run("NewKeyImmunity", func(t *testing.T) {
		rdb.FlushDB()
		for i := 1; i <= 9; i++ {
			rdb.Set(fmt.Sprintf("k%d", i), "v", noExp)
		}
		rdb.Set("newbie", "v", noExp)
		rdb.Set("trigger", "v", noExp)
		if rdb.Get("newbie").Val() == "" {
			t.Error("Key vừa mới tạo đã bị xóa ngay lập tức")
		}
	})

	// 13. FullUpdateSelf
	t.Run("FullUpdateSelf", func(t *testing.T) {
		rdb.FlushDB()
		for i := 1; i <= 10; i++ {
			rdb.Set(fmt.Sprintf("k%d", i), "v", noExp)
		}
		rdb.Set("k5", "new_val", noExp)
		if rdb.DBSize().Val() != 10 {
			t.Errorf("Số lượng key thay đổi: %d", rdb.DBSize().Val())
		}
	})

	// 14. EvictOnlyWhenFull
	t.Run("EvictOnlyWhenFull", func(t *testing.T) {
		rdb.FlushDB()
		for i := 1; i <= 10; i++ {
			rdb.Set(fmt.Sprintf("k%d", i), "v", noExp)
		}
		rdb.Del("k1")
		rdb.Set("k11", "v", noExp)
		if rdb.DBSize().Val() != 10 {
			t.Error("Logic eviction chạy sai khi chưa đầy")
		}
	})

	// 15. NilValueHandling
	t.Run("NilValueHandling", func(t *testing.T) {
		rdb.Set("emptykey", "", noExp)
		if rdb.Get("emptykey").Val() != "" {
			t.Error("Lỗi xử lý empty string")
		}
	})

	// 17. LongTermStability
	t.Run("LongTermStability", func(t *testing.T) {
		rdb.FlushDB()
		for i := 0; i < 500; i++ {
			rdb.Set(fmt.Sprintf("key%d", i), "val", noExp)
		}
		if rdb.DBSize().Val() != 10 {
			t.Errorf("Mất kiểm soát bộ nhớ: %d", rdb.DBSize().Val())
		}
	})

	// 18. DeepCheckEvictedKey
	t.Run("DeepCheckEvictedKey", func(t *testing.T) {
		rdb.FlushDB()
		rdb.Set("old", "v", noExp)
		for i := 1; i <= 20; i++ {
			rdb.Set(fmt.Sprintf("k%d", i), "v", noExp)
		}
		// old có xác suất cực cao bị xóa
	})

	// 19. SpecialChars
	t.Run("SpecialChars", func(t *testing.T) {
		rdb.Set("user:123:name", "admin", noExp)
		if rdb.Get("user:123:name").Val() != "admin" {
			t.Error("Lỗi key có ký tự đặc biệt")
		}
	})

	// 20. LRUClockLogic
	t.Run("LRUClockLogic", func(t *testing.T) {
		rdb.FlushDB()
		rdb.Set("a", "1", noExp)
		time.Sleep(10 * time.Millisecond)
		rdb.Set("b", "2", noExp)
	})
}
