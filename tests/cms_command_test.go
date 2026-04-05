package tests

import (
	"fmt"
	"testing"
)

// TODO : performance test cms with 100M element
// measure memory used
// check the error and rate
// TODO : review mean of error rate and ??
// TODO ưu nhược điểm CMS + so sánh với hyper log log
// có thể dùng thay thế cho nhau được ko
func TestCountMinSketchV6(t *testing.T) {
	rdb := SetupClientAndServer()

	// ==========================================
	// NHÓM 1: CMS.INITBYPROB
	// ==========================================
	t.Run("INITBYPROB", func(t *testing.T) {
		rdb.FlushDB()
		key := "cms:init"

		t.Run("TC-01_ValidInit", func(t *testing.T) {
			err := rdb.Do("CMS.INITBYPROB", key, 0.001, 0.01).Err()
			if err != nil {
				t.Fatalf("Lỗi: %v", err)
			}
		})

		t.Run("TC-02_DuplicateKey", func(t *testing.T) {
			err := rdb.Do("CMS.INITBYPROB", key, 0.001, 0.01).Err()
			if err == nil {
				t.Error("Mong đợi lỗi trùng key nhưng không thấy")
			}
		})

		t.Run("TC-03_ErrorZero", func(t *testing.T) {
			err := rdb.Do("CMS.INITBYPROB", "cms:e0", 0, 0.01).Err()
			if err == nil {
				t.Error("Lỗi logic: error=0")
			}
		})

		t.Run("TC-04_ErrorOne", func(t *testing.T) {
			err := rdb.Do("CMS.INITBYPROB", "cms:e1", 1.0, 0.01).Err()
			if err == nil {
				t.Error("Lỗi logic: error=1")
			}
		})

		t.Run("TC-05_ProbZero", func(t *testing.T) {
			err := rdb.Do("CMS.INITBYPROB", "cms:p0", 0.001, 0).Err()
			if err == nil {
				t.Error("Lỗi logic: prob=0")
			}
		})

		t.Run("TC-06_ProbOne", func(t *testing.T) {
			err := rdb.Do("CMS.INITBYPROB", "cms:p1", 0.001, 1.0).Err()
			if err == nil {
				t.Error("Lỗi logic: prob=1")
			}
		})

		t.Run("TC-07_InvalidArgs", func(t *testing.T) {
			err := rdb.Do("CMS.INITBYPROB", "cms:inv", "abc", "def").Err()
			if err == nil {
				t.Error("Mong đợi lỗi định dạng tham số")
			}
		})
	})

	// ==========================================
	// NHÓM 2: CMS.INCRBY
	// ==========================================
	t.Run("INCRBY", func(t *testing.T) {
		rdb.FlushDB()
		key := "cms:incr"
		rdb.Do("CMS.INITBYPROB", key, 0.01, 0.01)

		t.Run("TC-08_SingleItem", func(t *testing.T) {
			res, err := parseCMSResult(rdb.Do("CMS.INCRBY", key, "apple", 1).Result())
			if err != nil || res[0] != 1 {
				t.Errorf("Mong muốn [1], có %v", res)
			}
		})

		t.Run("TC-09_MultipleItems", func(t *testing.T) {
			res, err := parseCMSResult(rdb.Do("CMS.INCRBY", key, "banana", 2, "orange", 3).Result())
			if err != nil || len(res) != 2 || res[0] != 2 || res[1] != 3 {
				t.Errorf("Mong muốn [2 3], có %v", res)
			}
		})

		t.Run("TC-10_AutoCreate", func(t *testing.T) {
			res, err := parseCMSResult(rdb.Do("CMS.INCRBY", key, "grape", 5).Result())
			if err != nil || res[0] != 5 {
				t.Errorf("Auto-create failed: %v", res)
			}
		})

		t.Run("TC-11_NegativeIncr", func(t *testing.T) {
			err := rdb.Do("CMS.INCRBY", key, "apple", -1).Err()
			if err == nil {
				t.Error("Không báo lỗi khi tăng số âm")
			}
		})

		t.Run("TC-12_ZeroIncr", func(t *testing.T) {
			res, err := parseCMSResult(rdb.Do("CMS.INCRBY", key, "apple", 0).Result())
			if err != nil || res[0] < 1 {
				t.Error("Zero increment error")
			}
		})

		t.Run("TC-13_SpecialChars", func(t *testing.T) {
			res, err := parseCMSResult(rdb.Do("CMS.INCRBY", key, "", 10, "!@#$", 5).Result())
			if err != nil || res[0] != 10 {
				t.Error("Special chars failed")
			}
		})

		t.Run("TC-14_WrongType", func(t *testing.T) {
			rdb.Set("str_key", "value", 0)
			err := rdb.Do("CMS.INCRBY", "str_key", "apple", 1).Err()
			if err == nil {
				t.Error("Mong đợi WRONGTYPE")
			}
		})
	})

	// ==========================================
	// NHÓM 3: CMS.QUERY
	// ==========================================
	t.Run("QUERY", func(t *testing.T) {
		rdb.FlushDB()

		key := "cms:query"
		rdb.Do("CMS.INITBYPROB", key, 0.01, 0.01)
		rdb.Do("CMS.INCRBY", key, "cat", 10, "dog", 20)

		t.Run("TC-15_ExistingItem", func(t *testing.T) {
			res, err := parseCMSResult(rdb.Do("CMS.QUERY", key, "cat").Result())
			if err != nil || res[0] != 10 {
				t.Errorf("Lỗi query cat: %v", res)
			}
		})

		t.Run("TC-16_StrangerItem", func(t *testing.T) {
			res, err := parseCMSResult(rdb.Do("CMS.QUERY", key, "bird").Result())
			if err != nil || res[0] != 0 {
				t.Error("Bird nên là 0")
			}
		})

		t.Run("TC-17_MultipleItems", func(t *testing.T) {
			res, err := parseCMSResult(rdb.Do("CMS.QUERY", key, "cat", "dog", "fish").Result())
			if err != nil || res[0] != 10 || res[1] != 20 || res[2] != 0 {
				t.Errorf("Query nhiều item sai: %v", res)
			}
		})

		t.Run("TC-18_MissingKey", func(t *testing.T) {
			err := rdb.Do("CMS.QUERY", "missing_cms", "cat").Err()
			if err == nil {
				t.Error("Query key không tồn tại phải lỗi")
			}
		})

		t.Run("TC-19_WrongType", func(t *testing.T) {
			rdb.Set("str_key2", "value", 0)
			err := rdb.Do("CMS.QUERY", "str_key2", "cat").Err()
			if err == nil {
				t.Error("Mong đợi WRONGTYPE")
			}
		})

		t.Run("TC-20_CollisionCheck", func(t *testing.T) {
			ckey := "cms:coll"
			rdb.Do("CMS.INITBYPROB", ckey, 0.1, 0.1)
			rdb.Do("CMS.INCRBY", ckey, "target", 5)

			// Gây nhiễu
			for i := 0; i < 50; i++ {
				rdb.Do("CMS.INCRBY", ckey, fmt.Sprintf("n_%d", i), 1)
			}

			res, _ := parseCMSResult(rdb.Do("CMS.QUERY", ckey, "target").Result())
			if len(res) == 0 || res[0] < 5 {
				t.Errorf("CMS sai số quá mức: %v", res)
			}
		})
	})
}

// Helper để parse kết quả từ CMS (v6 trả về []interface{})
func parseCMSResult(res interface{}, err error) ([]int64, error) {
	if err != nil {
		return nil, err
	}
	arr, ok := res.([]interface{})
	if !ok {
		return nil, fmt.Errorf("không thể ép kiểu: %T", res)
	}
	var parsed []int64
	for _, item := range arr {
		if v, ok := item.(int64); ok {
			parsed = append(parsed, v)
		} else {
			return nil, fmt.Errorf("phần tử không phải int64: %T", item)
		}
	}
	return parsed, nil
}
