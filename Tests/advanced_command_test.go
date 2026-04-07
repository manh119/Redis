package Tests

import (
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

func TestRedisAdvancedCommands(t *testing.T) {
	rdb := SetupClientAndServer()

	// --- NHÓM TEST EXISTS ---
	t.Run("EXISTS", func(t *testing.T) {
		// Dọn dẹp database trước khi test nhóm này
		rdb.FlushDB()

		tests := []struct {
			name     string
			setup    func()
			keys     []string
			expected int64
		}{
			{"Key không tồn tại", func() {}, []string{"non_existent"}, 0},
			{"Một key tồn tại", func() { rdb.Set("k1", "v", 0) }, []string{"k1"}, 1},
			{"Nhiều key hỗn hợp", func() {
				rdb.Set("k1", "v", 0)
				rdb.Set("k2", "v", 0)
			}, []string{"k1", "k2", "k3"}, 2},
			{"Key rỗng", func() { rdb.Set("", "v", 0) }, []string{""}, 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tt.setup()
				count, _ := rdb.Exists(tt.keys...).Result()
				assert.Equal(t, tt.expected, count, tt.name)
			})
		}
	})

	// --- NHÓM TEST DEL ---
	t.Run("DEL", func(t *testing.T) {
		rdb.FlushDB()

		t.Run("Xóa nhiều key và lặp lại", func(t *testing.T) {
			rdb.Set("del_a", "1", 0)
			rdb.Set("del_b", "2", 0)

			// Xóa 2 key tồn tại + 1 key không tồn tại + 1 key lặp lại
			n, err := rdb.Del("del_a", "del_b", "ghost", "del_a").Result()
			assert.NoError(t, err)
			assert.Equal(t, int64(2), n, "Chỉ tính số lượng key thực tế bị xóa")

			// Kiểm tra GET lại
			_, err = rdb.Get("del_a").Result()
			assert.Equal(t, redis.Nil, err)
		})
	})

	// --- NHÓM TEST EXPIRE & TTL ---
	t.Run("EXPIRE_TTL_LOGIC", func(t *testing.T) {
		rdb.FlushDB()

		// Test Case: TTL Đặc biệt (-1, -2)
		t.Run("TTL Special Values", func(t *testing.T) {
			rdb.Set("permanent", "v", 0)
			assert.Equal(t, int64(-1), int64(rdb.TTL("permanent").Val().Seconds()), "Key không có expire trả về -1")
			assert.Equal(t, int64(-2), int64(rdb.TTL("no_key").Val().Seconds()), "Key không tồn tại trả về -2")
		})

		// Test Case: Tự động xóa (Short lived)
		t.Run("Auto Deletion", func(t *testing.T) {
			key := "short_lived"
			rdb.Set(key, "v", 100*time.Millisecond)

			// Đợi lâu hơn TTL một chút
			time.Sleep(150 * time.Millisecond)

			exists := rdb.Exists(key).Val()
			assert.Equal(t, int64(0), exists, "Key phải bị xóa sau khi hết TTL")
		})

		// Test Case: Persist
		t.Run("Persist Command", func(t *testing.T) {
			rdb.Set("per", "v", 10*time.Second)
			ok, _ := rdb.Persist("per").Result()
			assert.True(t, ok)
			assert.Equal(t, -1, int(rdb.TTL("per").Val().Seconds()))
		})
	})

	// 20. Test Concurrent Access
	t.Run("TestConcurrency", func(t *testing.T) {
		wg := sync.WaitGroup{}
		for i := 0; i < 1000; i++ {
			wg.Add(1)
			go func(val int) {
				defer wg.Done()
				rdb.Set("counter", val, 0)
				rdb.Get("counter")
			}(i)
		}
		wg.Wait()
	})
}
