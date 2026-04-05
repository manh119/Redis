package command_tests

import (
	"reflect"
	"sort"
	"testing"
)

func TestRedisSetCommands(t *testing.T) {
	rdb := SetupClientAndServer()

	// Helper để dọn dẹp sau mỗi test case
	flush := func() { rdb.FlushDB() }

	// --- NHÓM 1: SADD (1-5) ---
	t.Run("SADD_NewMember", func(t *testing.T) {
		flush()
		n, _ := rdb.SAdd("myset", "a").Result()
		if n != 1 {
			t.Errorf("SADD nên trả về 1 cho member mới, nhận: %d", n)
		}
	})

	t.Run("SADD_MultipleMembers", func(t *testing.T) {
		flush()
		n, _ := rdb.SAdd("myset", "a", "b", "c").Result()
		if n != 3 {
			t.Errorf("SADD nên trả về 3, nhận: %d", n)
		}
	})

	t.Run("SADD_DuplicateMember", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a")
		n, _ := rdb.SAdd("myset", "a").Result()
		if n != 0 {
			t.Errorf("SADD nên trả về 0 khi thêm phần tử đã tồn tại, nhận: %d", n)
		}
	})

	t.Run("SADD_MixNewAndOld", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a")
		n, _ := rdb.SAdd("myset", "a", "b").Result()
		if n != 1 {
			t.Errorf("SADD nên trả về 1 (chỉ tính 'b'), nhận: %d", n)
		}
	})

	t.Run("SADD_OnExistingKeyNotSet", func(t *testing.T) {
		flush()
		rdb.Set("notaset", "value", 0)
		err := rdb.SAdd("notaset", "a").Err()
		if err == nil {
			t.Error("SADD nên báo lỗi khi key không phải kiểu SET")
		}
	})

	// --- NHÓM 2: SISMEMBER (6-9) ---
	t.Run("SISMEMBER_Exists", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a")
		exists, _ := rdb.SIsMember("myset", "a").Result()
		if !exists {
			t.Error("SISMEMBER nên trả về true cho 'a'")
		}
	})

	t.Run("SISMEMBER_NotExists", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a")
		exists, _ := rdb.SIsMember("myset", "b").Result()
		if exists {
			t.Error("SISMEMBER nên trả về false cho 'b'")
		}
	})

	t.Run("SISMEMBER_NonExistentKey", func(t *testing.T) {
		flush()
		exists, _ := rdb.SIsMember("unknown", "a").Result()
		if exists {
			t.Error("SISMEMBER nên trả về false cho key không tồn tại")
		}
	})

	t.Run("SISMEMBER_EmptyString", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "")
		exists, _ := rdb.SIsMember("myset", "").Result()
		if !exists {
			t.Error("SISMEMBER nên hỗ trợ chuỗi rỗng")
		}
	})

	// --- NHÓM 3: SREM (10-14) ---
	t.Run("SREM_ExistingMember", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a", "b")
		n, _ := rdb.SRem("myset", "a").Result()
		if n != 1 {
			t.Errorf("SREM nên trả về 1, nhận: %d", n)
		}
	})

	t.Run("SREM_NonExistentMember", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a")
		n, _ := rdb.SRem("myset", "c").Result()
		if n != 0 {
			t.Errorf("SREM nên trả về 0 cho member không tồn tại, nhận: %d", n)
		}
	})

	t.Run("SREM_MultipleMembers", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a", "b", "c")
		n, _ := rdb.SRem("myset", "a", "c", "z").Result()
		if n != 2 {
			t.Errorf("SREM nên trả về 2 (a và c), nhận: %d", n)
		}
	})

	t.Run("SREM_LastMemberRemovesKey", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a")
		rdb.SRem("myset", "a")
		exists, _ := rdb.Exists("myset").Result()
		if exists == 1 {
			t.Error("Key nên bị xóa sau khi xóa member cuối cùng")
		}
	})

	t.Run("SREM_NonExistentKey", func(t *testing.T) {
		flush()
		n, _ := rdb.SRem("none", "a").Result()
		if n != 0 {
			t.Error("SREM trên key không tồn tại nên trả về 0")
		}
	})

	// --- NHÓM 4: SMEMBERS (15-20) ---
	t.Run("SMEMBERS_Basic", func(t *testing.T) {
		flush()
		members := []string{"a", "b"}
		rdb.SAdd("myset", "a", "b")
		res, _ := rdb.SMembers("myset").Result()
		sort.Strings(res)
		sort.Strings(members)
		if !reflect.DeepEqual(res, members) {
			t.Errorf("SMEMBERS sai. Nhận: %v", res)
		}
	})

	t.Run("SMEMBERS_NonExistentKey", func(t *testing.T) {
		flush()
		res, _ := rdb.SMembers("ghost").Result()
		if len(res) != 0 {
			t.Error("SMEMBERS trên key trống nên trả về mảng rỗng")
		}
	})

	t.Run("SMEMBERS_AfterSREM", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "a", "b")
		rdb.SRem("myset", "a")
		res, _ := rdb.SMembers("myset").Result()
		if len(res) != 1 || res[0] != "b" {
			t.Error("Dữ liệu SMEMBERS không đồng bộ sau SREM")
		}
	})

	t.Run("SMEMBERS_LargeSet", func(t *testing.T) {
		flush()
		for i := 0; i < 100; i++ {
			rdb.SAdd("bigset", i)
		}
		res, _ := rdb.SMembers("bigset").Result()
		if len(res) != 100 {
			t.Errorf("Thiếu phần tử trong bigset, có: %d", len(res))
		}
	})

	t.Run("SADD_CaseSensitivity", func(t *testing.T) {
		flush()
		rdb.SAdd("myset", "Apple", "apple")
		res, _ := rdb.SMembers("myset").Result()
		if len(res) != 2 {
			t.Error("Set nên phân biệt chữ hoa chữ thường")
		}
	})

	t.Run("SADD_SpecialChars", func(t *testing.T) {
		flush()
		val := "!@#$%^&*()"
		rdb.SAdd("myset", val)
		exists, _ := rdb.SIsMember("myset", val).Result()
		if !exists {
			t.Error("Không xử lý được ký tự đặc biệt")
		}
	})
}
