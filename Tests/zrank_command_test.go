package Tests

import (
	"testing"

	"github.com/go-redis/redis"
)

func TestZRANK_Extended(t *testing.T) {
	rdb := SetupClientAndServer()
	flush := func() { rdb.FlushDB() }
	flush() // Giả sử hàm flush của bạn xóa sạch data trước mỗi test

	// Setup dữ liệu mẫu
	// Thứ tự mong muốn (Score tăng dần, sau đó là Lexicographical):
	// 0: a (Score 10)
	// 1: b (Score 20)
	// 2: c (Score 20) -> "c" > "b" nên rank cao hơn
	// 3: d (Score 30)
	rdb.ZAdd("myzset",
		redis.Z{Score: 10, Member: "a"},
		redis.Z{Score: 20, Member: "c"},
		redis.Z{Score: 20, Member: "b"},
		redis.Z{Score: 30, Member: "d"},
	)

	t.Run("ZRANK_FirstElement", func(t *testing.T) {
		rank, err := rdb.ZRank("myzset", "a").Result()
		if err != nil || rank != 0 {
			t.Errorf("Expected rank 0 for 'a', got %d, err: %v", rank, err)
		}
	})

	t.Run("ZRANK_LastElement", func(t *testing.T) {
		rank, err := rdb.ZRank("myzset", "d").Result()
		if err != nil || rank != 3 {
			t.Errorf("Expected rank 3 for 'd', got %d, err: %v", rank, err)
		}
	})

	t.Run("ZRANK_SameScore_Lexicographical", func(t *testing.T) {
		// "b" và "c" cùng score 20, "b" đứng trước "c" theo bảng chữ cái
		rankB, _ := rdb.ZRank("myzset", "b").Result()
		rankC, _ := rdb.ZRank("myzset", "c").Result()

		if rankB != 1 || rankC != 2 {
			t.Errorf("Lexicographical order failed: 'b' expected 1 (got %d), 'c' expected 2 (got %d)", rankB, rankC)
		}
	})

	t.Run("ZRANK_NonExistentMember", func(t *testing.T) {
		// Member không tồn tại trong Set phải trả về lỗi redis.Nil
		_, err := rdb.ZRank("myzset", "non-existent").Result()
		if err != redis.Nil {
			t.Errorf("Expected redis.Nil for missing member, got %v", err)
		}
	})

	t.Run("ZRANK_NonExistentKey", func(t *testing.T) {
		// Key không tồn tại phải trả về lỗi redis.Nil
		_, err := rdb.ZRank("ghost-key", "any").Result()
		if err != redis.Nil {
			t.Errorf("Expected redis.Nil for non-existent key, got %v", err)
		}
	})

	t.Run("ZRANK_UpdateScore_ChangesRank", func(t *testing.T) {
		// Thử cập nhật 'a' từ score 10 lên 100 để nó nhảy xuống cuối hàng
		rdb.ZAdd("myzset", redis.Z{Score: 100, Member: "a"})
		rank, _ := rdb.ZRank("myzset", "a").Result()
		if rank != 3 {
			t.Errorf("Rank should be 3 after score update, got %d", rank)
		}
	})
}
