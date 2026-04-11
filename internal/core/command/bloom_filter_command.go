package command

import (
	"fmt"
	"strconv"

	"github.com/manh119/Redis/internal/core"
	"github.com/manh119/Redis/internal/core/data_structure"
)

// ---------------- BLOOM FILTER -----------------------
// BF.RESERVE key error_rate capacity -> OK
// (error) ERR item exists
func HandleRESERVE(cmd *Command) (string, error) {
	argCount := len(cmd.Args)
	if argCount != 3 {
		return "", fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.Args[0]
	_, exist := storage.BloomFilterStore[key]
	if exist {
		return "", fmt.Errorf("BLOOM FILTER %s already exist", key)
	}

	errorRate, err := strconv.ParseFloat(cmd.Args[1], 64)
	if err != nil || errorRate <= 0 || errorRate >= 1 {
		return "", fmt.Errorf("Error rate is not valid value", key)
	}

	entries, err := strconv.ParseUint(cmd.Args[2], 10, 64)
	if err != nil || entries == 0 {
		return "", fmt.Errorf("Entries is not valid value", key)
	}

	storage.BloomFilterStore[key] = data_structure.NewBloomFilter(errorRate, entries)
	return "OK", nil
}

// BF.MADD key item
// redis> BF.MADD bf item1 item2 item2
// 1) (integer) 1
// 2) (integer) 1
// 3) (integer) 0
func HandleMADD(cmd *Command) ([]uint64, error) {
	argCount := len(cmd.Args)
	if argCount < 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.Args[0]

	_, err := storage.DictStore.Get(key)
	if err == nil {
		return nil, fmt.Errorf("SET %s already exists, wrong type for add bloom filter", key)
	}

	bf, exist := storage.BloomFilterStore[key]
	if !exist {
		bf = data_structure.NewBloomFilter(0.01, 100)
		storage.BloomFilterStore[key] = bf
	}

	counts := make([]uint64, argCount-1)
	for i := 1; i < argCount; i++ {
		if bf.Exist(cmd.Args[i]) {
			counts[i-1] = 0
		} else {
			counts[i-1] = 1
		}
		bf.Add(cmd.Args[i])
	}

	return counts, nil
}

// BF.MEXISTS key item
// redis> BF.MADD bf item1 item2
// 1) (integer) 1
// 2) (integer) 1
func HandleMEXISTS(cmd *Command) ([]uint64, error) {
	argCount := len(cmd.Args)
	if argCount < 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.Args[0]
	bf, exist := storage.BloomFilterStore[key]
	counts := make([]uint64, argCount-1)
	if !exist {
		return counts, nil
	}

	for i := 1; i < argCount; i++ {
		if bf.Exist(cmd.Args[i]) {
			counts[i-1] = 1
		} else {
			counts[i-1] = 0
		}
	}

	return counts, nil
}
