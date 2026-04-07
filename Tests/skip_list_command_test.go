package Tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis"
)

func TestRedisSortedSetAdvanced(t *testing.T) {
	rdb := SetupClientAndServer()
	flush := func() { rdb.FlushDB() }

	// --- NHÓM 1: ZADD (Thêm mới & Cập nhật) ---

	t.Run("1_ZADD_Single", func(t *testing.T) {
		flush()
		n, _ := rdb.ZAdd("z", redis.Z{Score: 1, Member: "a"}).Result()
		if n != 1 {
			t.Errorf("Nên trả về 1, nhận: %d", n)
		}
	})

	t.Run("2_ZADD_Multi", func(t *testing.T) {
		flush()
		n, _ := rdb.ZAdd("z", redis.Z{Score: 1, Member: "a"}, redis.Z{Score: 2, Member: "b"}).Result()
		if n != 2 {
			t.Errorf("Nên trả về 2, nhận: %d", n)
		}
	})

	t.Run("3_ZADD_UpdateScore", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 1, Member: "a"})
		n, _ := rdb.ZAdd("z", redis.Z{Score: 5, Member: "a"}).Result()
		// n = 0 vì member đã tồn tại, chỉ cập nhật score
		if n != 0 {
			t.Errorf("Cập nhật score không nên tính là member mới")
		}
		s, _ := rdb.ZScore("z", "a").Result()
		time.Sleep(1000 * time.Millisecond)
		if s != 5 {
			t.Errorf("Score chưa được cập nhật")
		}
	})

	t.Run("4_ZADD_FloatScore", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 3.14, Member: "pi"})
		s, _ := rdb.ZScore("z", "pi").Result()
		if s != 3.14 {
			t.Errorf("Lỗi lưu trữ số thực")
		}
	})

	t.Run("4_ZADD_IntScore", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 3, Member: "pi"})
		s, _ := rdb.ZScore("z", "pi").Result()
		if s != 3 {
			t.Errorf("Lỗi lưu trữ số int")
		}
	})

	// --- NHÓM 2: ZSCORE & ZCARD ---

	t.Run("5_ZSCORE_NonExistentMember", func(t *testing.T) {
		flush()
		_, err := rdb.ZScore("z", "ghost").Result()
		if err != redis.Nil {
			t.Errorf("Nên trả về lỗi redis.Nil")
		}
	})

	t.Run("6_ZCARD_EmptyKey", func(t *testing.T) {
		flush()
		n, _ := rdb.ZCard("empty").Result()
		if n != 0 {
			t.Errorf("Key trống nên có Card = 0")
		}
	})

	// --- NHÓM 3: ZRANK & ZREVRANK (Thứ tự) ---

	t.Run("7_ZRANK_Basic", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 10, Member: "min"}, redis.Z{Score: 20, Member: "max"})
		rank, _ := rdb.ZRank("z", "max").Result()
		if rank != 1 {
			t.Errorf("Rank phải là 1 (0-indexed)")
		}
	})

	t.Run("8_ZREVRANK_Leaderboard", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 10, Member: "p1"}, redis.Z{Score: 50, Member: "top"})
		rank, _ := rdb.ZRevRank("z", "top").Result()
		if rank != 0 {
			t.Errorf("Top score phải có RevRank = 0")
		}
	})

	t.Run("9_ZRANK_SameScore_Lex", func(t *testing.T) {
		flush()
		// Cùng score, sắp xếp theo tên (a < b)
		rdb.ZAdd("z", redis.Z{Score: 10, Member: "b"}, redis.Z{Score: 10, Member: "a"})
		rank, _ := rdb.ZRank("z", "a").Result()
		if rank != 0 {
			t.Errorf("Cùng score, 'a' phải đứng trước 'b'")
		}
	})

	// --- NHÓM 4: ZREM & ZINCRBY ---

	t.Run("10_ZREM_Single", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 1, Member: "a"})
		n, _ := rdb.ZRem("z", "a").Result()
		if n != 1 {
			t.Errorf("Xóa thất bại")
		}
	})

	t.Run("11_ZREM_Multi", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 1, Member: "a"}, redis.Z{Score: 2, Member: "b"})
		n, _ := rdb.ZRem("z", "a", "b", "c").Result()
		if n != 2 {
			t.Errorf("Nên xóa được 2 (c không tồn tại)")
		}
	})

	t.Run("12_ZINCRBY_Positive", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 10, Member: "a"})
		newS, _ := rdb.ZIncrBy("z", 5.5, "a").Result()
		if newS != 15.5 {
			t.Errorf("IncrBy sai kết quả")
		}
	})

	t.Run("13_ZINCRBY_Negative", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 10, Member: "a"})
		newS, _ := rdb.ZIncrBy("z", -5, "a").Result()
		if newS != 5 {
			t.Errorf("Giảm điểm thất bại")
		}
	})

	// --- NHÓM 5: ZRANGE (Lấy danh sách) ---

	t.Run("14_ZRANGE_All", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 1, Member: "a"}, redis.Z{Score: 2, Member: "b"})
		res, _ := rdb.ZRange("z", 0, -1).Result()
		if len(res) != 2 {
			t.Errorf("Lấy toàn bộ danh sách thất bại")
		}
	})

	t.Run("15_ZRANGE_WithScores", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 10, Member: "a"})
		res, _ := rdb.ZRangeWithScores("z", 0, 0).Result()
		if res[0].Score != 10 {
			t.Errorf("Lấy kèm score bị sai")
		}
	})

	t.Run("16_ZREVRANGE_Top3", func(t *testing.T) {
		flush()
		for i := 1; i <= 5; i++ {
			rdb.ZAdd("z", redis.Z{Score: float64(i), Member: fmt.Sprintf("p%d", i)})
		}
		res, _ := rdb.ZRevRange("z", 0, 2).Result()
		if res[0] != "p5" || len(res) != 3 {
			t.Errorf("Lấy Top 3 sai")
		}
	})

	// --- NHÓM 6: TRƯỜNG HỢP BIÊN (Edge Cases) ---

	t.Run("17_ZRANK_KeyNotFound", func(t *testing.T) {
		flush()
		_, err := rdb.ZRank("no_key", "a").Result()
		if err != redis.Nil {
			t.Errorf("Key không tồn tại phải trả về redis.Nil")
		}
	})

	t.Run("18_ZADD_OverwritingType", func(t *testing.T) {
		flush()
		rdb.Set("not_a_zset", "hello", 0)
		_, err := rdb.ZAdd("not_a_zset", redis.Z{Score: 1, Member: "a"}).Result()
		if err == nil {
			t.Errorf("Phải báo lỗi khi thao tác ZADD lên kiểu dữ liệu String")
		}
	})

	t.Run("19_ZRANGE_OutofBounds", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 1, Member: "a"})
		res, _ := rdb.ZRange("z", 10, 20).Result()
		if len(res) != 0 {
			t.Errorf("Range ngoài phạm vi phải trả về slice rỗng")
		}
	})

	t.Run("20_ZSCORE_CaseSensitive", func(t *testing.T) {
		flush()
		rdb.ZAdd("z", redis.Z{Score: 100, Member: "Player"})
		_, err := rdb.ZScore("z", "player").Result()
		if err != redis.Nil {
			t.Errorf("Member trong Redis phải phân biệt chữ hoa chữ thường")
		}
	})
}
