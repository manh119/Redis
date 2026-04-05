package command_tests

//
//func TestBloomFilterPerformance(t *testing.T) {
//	rdb := SetupClientAndServer()
//
//	// ==========================================
//	// NHÓM 5: PERFORMANCE & SCALABILITY
//	// ==========================================
//	t.Run("PERFORMANCE", func(t *testing.T) {
//		rdb.FlushDB()
//
//		t.Run("TC-21_HighThroughput_MAdd", func(t *testing.T) {
//			key := "{bf}:perf:madd"
//			start := time.Now()
//			numItems := 10000
//
//			// Giả lập thêm 10,000 phần tử bằng batching
//			for i := 0; i < 100; i++ {
//				items := make([]interface{}, 101)
//				items[0] = key
//				for j := 1; j <= 100; j++ {
//					items[j] = fmt.Sprintf("item-%d-%d", i, j)
//				}
//				rdb.Do("BF.MADD", items...)
//			}
//
//			duration := time.Since(start)
//			t.Logf("MADD 10,000 items tốn: %v", duration)
//			if duration > 2*time.Second {
//				t.Errorf("Performance MADD quá chậm: %v", duration)
//			}
//		})
//
//		t.Run("TC-22_Latency_MExists_LargeSet", func(t *testing.T) {
//			key := "{bf}:perf:mexists"
//			// Thêm sẵn 5000 phần tử
//			for i := 0; i < 50; i++ {
//				rdb.Do("BF.MADD", key, i, i+1, i+2 /*...*/)
//			}
//
//			start := time.Now()
//			// Kiểm tra 1000 phần tử ngẫu nhiên xem có tồn tại không
//			for i := 0; i < 1000; i++ {
//				rdb.Do("BF.MEXISTS", key, fmt.Sprintf("item-%d", i))
//			}
//			duration := time.Since(start)
//			t.Logf("1,000 lần gọi MEXISTS đơn lẻ tốn: %v", duration)
//		})
//
//		t.Run("TC-23_Memory_Consumption_Check", func(t *testing.T) {
//			// So sánh memory giữa filter dung lượng thấp và cao
//			rdb.Do("BF.RESERVE", "{bf}:small", 0.1, 100)
//			rdb.Do("BF.RESERVE", "{bf}:large", 0.0001, 100000)
//
//			infoSmall, _ := rdb.Do("BF.INFO", "{bf}:small").Result()
//			infoLarge, _ := rdb.Do("BF.INFO", "{bf}:large").Result()
//
//			t.Logf("Info Small: %v", infoSmall)
//			t.Logf("Info Large: %v", infoLarge)
//			// Kiểm tra dung lượng (Size) trong infoLarge phải lớn hơn nhiều infoSmall
//		})
//
//		t.Run("TC-24_Pipeline_Efficiency", func(t *testing.T) {
//			key := "{bf}:pipeline"
//			pipe := rdb.Pipeline()
//			start := time.Now()
//
//			for i := 0; i < 1000; i++ {
//				pipe.Do("BF.MADD", key, i)
//			}
//			_, err := pipe.Exec()
//
//			duration := time.Since(start)
//			if err != nil {
//				t.Fatalf("Pipeline lỗi: %v", err)
//			}
//			t.Logf("Pipeline 1,000 lệnh MADD tốn: %v", duration)
//		})
//
//		t.Run("TC-25_FalsePositive_Rate_Verification", func(t *testing.T) {
//			key := "{bf}:fpr"
//			capacity := 10000
//			errorRate := 0.01 // 1%
//			rdb.Do("BF.RESERVE", key, errorRate, capacity)
//
//			// 1. Fill data đến đúng capacity
//			for i := 0; i < capacity; i++ {
//				rdb.Do("BF.ADD", key, fmt.Sprintf("exist-%d", i))
//			}
//
//			// 2. Check 1000 phần tử chắc chắn KHÔNG tồn tại
//			falsePositives := 0
//			testCount := 1000
//			for i := 0; i < testCount; i++ {
//				res, _ := rdb.Do("BF.EXISTS", key, fmt.Sprintf("not-exist-%d", i)).Int()
//				if res == 1 {
//					falsePositives++
//				}
//			}
//
//			actualRate := float64(falsePositives) / float64(testCount)
//			t.Logf("False Positive Rate thực tế: %f (Kỳ vọng: %f)", actualRate, errorRate)
//			if actualRate > errorRate*1.5 { // Cho phép sai số nhẹ
//				t.Errorf("Tỷ lệ False Positive vượt quá ngưỡng cho phép: %f", actualRate)
//			}
//		})
//
//		t.Run("TC-26_Scaling_Performance_Degradation", func(t *testing.T) {
//			// Kiểm tra xem khi Bloom Filter tự động mở rộng (auto-scaling),
//			// latency có bị tăng đột biến không
//			key := "{bf}:auto-scale"
//			rdb.Do("BF.RESERVE", key, 0.01, 100) // Capacity nhỏ để ép scaling nhanh
//
//			latencies := []time.Duration{}
//			for i := 0; i < 5000; i++ {
//				s := time.Now()
//				rdb.Do("BF.ADD", key, i)
//				latencies = append(latencies, time.Since(s))
//			}
//			// Tính toán hoặc log ra các điểm latency cao nhất (thường là lúc tạo sub-filter mới)
//			t.Log("Hoàn tất test auto-scaling latency")
//		})
//
//		t.Run("TC-27_Concurrent_Access", func(t *testing.T) {
//			key := "{bf}:concurrent"
//			rdb.Do("BF.RESERVE", key, 0.01, 10000)
//
//			var wg sync.WaitGroup
//			workers := 10
//			itemsPerWorker := 100
//
//			for w := 0; w < workers; w++ {
//				wg.Add(1)
//				go func(workerID int) {
//					defer wg.Done()
//					for i := 0; i < itemsPerWorker; i++ {
//						rdb.Do("BF.MADD", key, fmt.Sprintf("w%d-i%d", workerID, i))
//					}
//				}(w)
//			}
//			wg.Wait()
//			// Kiểm tra tổng số lượng item (dùng BF.INFO)
//		})
//
//		t.Run("TC-28_Large_Item_Payload", func(t *testing.T) {
//			// Test xem item có kích thước lớn (chuỗi dài) có làm chậm hash không
//			key := "{bf}:large-item"
//			largeItem := strings.Repeat("a", 1024*10) // 10KB string
//
//			start := time.Now()
//			rdb.Do("BF.ADD", key, largeItem)
//			duration := time.Since(start)
//
//			if duration > 100*time.Millisecond {
//				t.Errorf("Xử lý item lớn quá chậm: %v", duration)
//			}
//		})
//
//		t.Run("TC-29_Bulk_MEXISTS_Performance", func(t *testing.T) {
//			// So sánh 1000 lần gọi EXISTS đơn lẻ vs 1 lần gọi MEXISTS 1000 items
//			key := "{bf}:bulk-exists"
//			items := make([]interface{}, 1001)
//			items[0] = key
//			for i := 1; i <= 1000; i++ {
//				items[i] = i
//			}
//			rdb.Do("BF.MADD", items...)
//
//			// Đo thời gian MEXISTS
//			start := time.Now()
//			rdb.Do("BF.MEXISTS", items...)
//			t.Logf("Bulk MEXISTS (1000 items) tốn: %v", time.Since(start))
//		})
//
//		t.Run("TC-30_Memory_Fragmentation_After_Flush", func(t *testing.T) {
//			// Kiểm tra giải phóng bộ nhớ sau khi xóa các Bloom Filter lớn
//			beforeMem := getRedisMemory(rdb)
//
//			for i := 0; i < 10; i++ {
//				rdb.Do("BF.RESERVE", fmt.Sprintf("{bf}:mem-%d", i), 0.0001, 100000)
//			}
//			rdb.FlushDB()
//
//			afterMem := getRedisMemory(rdb)
//			t.Logf("Memory trước: %d, sau khi Flush: %d", beforeMem, afterMem)
//			// afterMem nên gần bằng beforeMem
//		})
//	})
//}
//
//// Hàm hỗ trợ lấy memory của Redis
//func getRedisMemory(rdb *redis.Client) int64 {
//	res, _ := rdb.Info("memory").Result()
//	// Parse chuỗi "used_memory:..." để lấy số (logic đơn giản hóa)
//	return 0
//}
