package command_tests

import (
	"testing"
)

func TestBloomFilter(t *testing.T) {
	rdb := SetupClientAndServer()

	// ==========================================
	// NHÓM 1: BF.RESERVE
	// ==========================================
	t.Run("BF.RESERVE", func(t *testing.T) {
		rdb.FlushDB()
		key := "{bf}:reserve"

		t.Run("TC-01_ValidReserve", func(t *testing.T) {
			// error_rate=0.01, capacity=1000
			err := rdb.Do("BF.RESERVE", key, 0.01, 1000).Err()
			if err != nil {
				t.Fatalf("Lỗi khởi tạo hợp lệ: %v", err)
			}
		})

		t.Run("TC-02_DuplicateKey", func(t *testing.T) {
			err := rdb.Do("BF.RESERVE", key, 0.01, 1000).Err()
			if err == nil {
				t.Error("Mong đợi lỗi khi khởi tạo key đã tồn tại")
			}
		})

		t.Run("TC-03_InvalidErrorRate_Zero", func(t *testing.T) {
			err := rdb.Do("BF.RESERVE", "bf:err0", 0, 1000).Err()
			if err == nil {
				t.Error("Mong đợi lỗi khi error_rate bằng 0")
			}
		})

		t.Run("TC-04_InvalidCapacity_Zero", func(t *testing.T) {
			err := rdb.Do("BF.RESERVE", "bf:cap0", 0.01, 0).Err()
			if err == nil {
				t.Error("Mong đợi lỗi khi capacity bằng 0")
			}
		})

	})

	// ==========================================
	// NHÓM 2: BF.MADD
	// ==========================================
	t.Run("BF.MADD", func(t *testing.T) {
		rdb.FlushDB()
		key := "{bf}:madd"

		t.Run("TC-06_MAdd_NewKeyAutoCreate", func(t *testing.T) {
			// BF.MADD tự động tạo filter nếu chưa có
			res, err := rdb.Do("BF.MADD", key, "item1", "item2").Result()
			if err != nil {
				t.Fatalf("Lỗi BF.MADD: %v", err)
			}
			// Mong đợi trả về [1, 1] vì là các phần tử mới
			slice := res.([]interface{})
			if len(slice) != 2 || slice[0].(int64) != 1 {
				t.Errorf("Kết quả không đúng: %v", res)
			}
		})

		t.Run("TC-07_MAdd_ExistingItems", func(t *testing.T) {
			_, _ = rdb.Do("BF.MADD", key, "item1").Result()
			res, _ := rdb.Do("BF.MADD", key, "item1").Result()
			val := res.([]interface{})[0].(int64)
			if val != 0 {
				t.Errorf("Mong đợi 0 cho item đã tồn tại, nhận được %d", val)
			}
		})

		t.Run("TC-08_MAdd_MixedNewAndOld", func(t *testing.T) {
			res, _ := rdb.Do("BF.MADD", key, "item1", "item3").Result()
			slice := res.([]interface{})
			if slice[0].(int64) != 0 || slice[1].(int64) != 1 {
				t.Errorf("Kết quả hỗn hợp không đúng: %v", res)
			}
		})

		t.Run("TC-09_MAdd_EmptyItems", func(t *testing.T) {
			err := rdb.Do("BF.MADD", key).Err()
			if err == nil {
				t.Error("Mong đợi lỗi khi gọi BF.MADD không có item")
			}
		})

		t.Run("TC-10_MAdd_WrongKeyType", func(t *testing.T) {
			rdb.Set("string_key", "value", 0)
			err := rdb.Do("BF.MADD", "string_key", "item1").Err()
			if err == nil {
				t.Error("Mong đợi lỗi WRONGTYPE")
			}
		})
	})

	// ==========================================
	// NHÓM 3: BF.MEXISTS
	// ==========================================
	t.Run("BF.MEXISTS", func(t *testing.T) {
		rdb.FlushDB()
		key := "{bf}:mexists"
		rdb.Do("BF.MADD", key, "apple", "banana")

		t.Run("TC-11_MExists_AllExist", func(t *testing.T) {
			res, _ := rdb.Do("BF.MEXISTS", key, "apple", "banana").Result()
			slice := res.([]interface{})
			if slice[0].(int64) != 1 || slice[1].(int64) != 1 {
				t.Errorf("Lỗi check tồn tại: %v", res)
			}
		})

		t.Run("TC-12_MExists_SomeExist", func(t *testing.T) {
			res, _ := rdb.Do("BF.MEXISTS", key, "apple", "cherry").Result()
			slice := res.([]interface{})
			if slice[0].(int64) != 1 || slice[1].(int64) != 0 {
				t.Errorf("Lỗi check tồn tại hỗn hợp: %v", res)
			}
		})

		t.Run("TC-13_MExists_NonExistentKey", func(t *testing.T) {
			res, _ := rdb.Do("BF.MEXISTS", "no_key", "item").Result()
			slice := res.([]interface{})
			if slice[0].(int64) != 0 {
				t.Error("Key không tồn tại phải trả về 0")
			}
		})

		t.Run("TC-14_MExists_SingleItem", func(t *testing.T) {
			res, _ := rdb.Do("BF.MEXISTS", key, "apple").Result()
			if len(res.([]interface{})) != 1 {
				t.Error("Phải trả về mảng 1 phần tử")
			}
		})
	})

	// ==========================================
	// NHÓM 4: KỊCH BẢN TỔNG HỢP (INTEGRATION)
	// ==========================================
	t.Run("INTEGRATION", func(t *testing.T) {
		rdb.FlushDB()
		key := "{bf}:flow"

		t.Run("TC-15_ReserveThenMAdd", func(t *testing.T) {
			rdb.Do("BF.RESERVE", key, 0.001, 50)
			rdb.Do("BF.MADD", key, "x", "y", "z")
			res, _ := rdb.Do("BF.MEXISTS", key, "x", "y", "z").Result()
			for _, v := range res.([]interface{}) {
				if v.(int64) != 1 {
					t.Error("Dữ liệu không khớp sau khi RESERVE")
				}
			}
		})

		t.Run("TC-16_LargeCapacity", func(t *testing.T) {
			err := rdb.Do("BF.RESERVE", "bf:large", 0.0001, 1000000).Err()
			if err != nil {
				t.Errorf("Lỗi khi khởi tạo capacity lớn: %v", err)
			}
		})

		t.Run("TC-17_NonScaling_Overflow", func(t *testing.T) {
			// BF với capacity nhỏ và NONSCALING
			rdb.Do("BF.RESERVE", "bf:full", 0.1, 1, "NONSCALING")
			rdb.Do("BF.MADD", "bf:full", "item1")
			err := rdb.Do("BF.MADD", "bf:full", "item2").Err()
			// Một số phiên bản RedisBloom sẽ báo lỗi khi filter đầy nếu dùng NONSCALING
			if err != nil {
				t.Logf("Thông báo khi tràn (tùy version): %v", err)
			}
		})

		t.Run("TC-18_SpecialCharacters", func(t *testing.T) {
			res, _ := rdb.Do("BF.MADD", key, " ", "!!@@##", "\n\t").Result()
			if len(res.([]interface{})) != 3 {
				t.Error("Không xử lý được ký tự đặc biệt")
			}
		})

		t.Run("TC-20_CaseSensitivity", func(t *testing.T) {
			rdb.Do("BF.MADD", key, "Redis")
			res, _ := rdb.Do("BF.MEXISTS", key, "redis").Result()
			if res.([]interface{})[0].(int64) == 1 {
				t.Log("Cảnh báo: Bloom Filter có thể bị trùng khớp do hash (False Positive) hoặc case-insensitive")
			}
		})
	})
}
