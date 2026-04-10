package command

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/manh119/Redis/internal/config"
	"github.com/manh119/Redis/internal/core"
	"github.com/manh119/Redis/internal/core/data_structure"
)

// ZADD myzset score(float64) member -> 1
func HandleZADD(cmd *Command) (int, error) {
	argCount := len(cmd.args)
	if argCount < 3 || (argCount-1)%2 != 0 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	skipList, exist := storage.SkipListStore[key]
	if !exist {
		skipList = data_structure.NewSkipList()
	}

	added := 0
	for i := 1; i < argCount; i = i + 2 {
		score, err := strconv.ParseFloat(cmd.args[i], 64)
		if err != nil {
			return 0, errors.New("value is not a valid float")
		}
		value := cmd.args[i+1]

		// case key already exist in skip list -> delete old node + insert new
		_, err = skipList.GetScore(value)
		if err == nil {
			skipList.Delete(value)
			skipList.Insert(value, score)
		} else {
			added++
			skipList.Insert(value, score)
		}

	}
	storage.SkipListStore[key] = skipList
	return added, nil
}

// ZSCORE zmyset member
func HandleZSCORE(cmd *Command) (any, error) {
	argCount := len(cmd.args)
	if argCount < 2 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	skipList, exist := storage.SkipListStore[key]
	if !exist {
		return nil, nil
	}
	value := cmd.args[1]
	score, err := skipList.GetScore(value)
	if err != nil {
		return nil, err
	}
	return float64(score), nil
}

// ZRANK myzset three -> 3
func HandleZRANK(cmd *Command) (any, error) {
	argCount := len(cmd.args)
	if argCount < 2 {
		return 0, fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}
	key := cmd.args[0]
	skipList, exist := storage.SkipListStore[key]
	if !exist {
		return config.NILL, nil
	}
	value := cmd.args[1]
	rank, err := skipList.GetRank(value)
	if err != nil || rank < 0 {
		return config.NILL, err
	}
	return rank, nil
}
