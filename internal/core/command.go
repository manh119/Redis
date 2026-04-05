package core

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/manh119/Redis/internal/core/constant"
	"github.com/manh119/Redis/internal/core/data_structure"
)

// example: set key value EX 100
// => cmd = set, args = ["key", "value", "EX", "100"]
type Command struct {
	Cmd  string
	args []string
}

func NewCommand(cmd string, args []string) *Command {
	return &Command{Cmd: cmd, args: args}
}

func HandleTTL(cmd *Command) (any, error) {
	if len(cmd.args) == 1 {
		key := cmd.args[0]
		return dictStore.Ttl(key), nil
	}
	return "", errors.New("invalid command")
}

func HandlePing(cmd *Command) (string, error) {
	if cmd.args == nil || len(cmd.args) == 0 {
		return "PONG", nil
	}

	if len(cmd.args) == 1 {
		return cmd.args[0], nil
	}

	return "", errors.New("invalid command")
}

func HandleGet(cmd *Command) (any, error) {
	if len(cmd.args) == 1 {
		key := cmd.args[0]
		return dictStore.Get(key), nil
	}
	return "", errors.New("invalid command")
}

// expire key 100s
func HandleExpire(cmd *Command) (any, error) {
	if len(cmd.args) == 2 {
		key := cmd.args[0]
		ttl := cmd.args[1]
		parsedTTL, err := strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		return dictStore.Expire(key, parsedTTL*1000), nil
	}
	return "", errors.New("invalid number of args")
}

// exists key1 key2 -> 2
func HandleExists(cmd *Command) (any, error) {
	if len(cmd.args) == 0 {
		return "", errors.New("invalid number of args")
	}
	return dictStore.Exists(cmd.args), nil
}

// del key1 key2
func HandleDel(cmd *Command) (any, error) {
	if len(cmd.args) == 0 {
		return "", errors.New("invalid number of args")
	}
	return dictStore.Del(cmd.args), nil
}

// set key value
// set key value EX ttl
func HandleSet(cmd *Command) (string, error) {
	argCount := len(cmd.args)
	if argCount != 2 && argCount != 4 {
		return "", errors.New("ERR wrong number of arguments for 'set' command")
	}
	key := cmd.args[0]
	value := cmd.args[1]
	var ttl int64 = -1
	if argCount == 4 {
		ttlStr := cmd.args[3]
		parsedTTL, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		ttl = parsedTTL
		if strings.ToUpper(cmd.args[2]) == "EX" {
			dictStore.Set(key, value, ttl*1000)
		} else if strings.ToUpper(cmd.args[2]) == "PX" { // ttl in miliSecond
			dictStore.Set(key, value, ttl)
		} else {
			return "", errors.New("ERR unknown command")
		}
	}
	dictStore.Set(key, value, ttl)
	return "OK", nil
}

// PERSIST key -> set ttl = -1 if key is valid
func HandlePERSIST(cmd *Command) (int, error) {
	argCount := len(cmd.args)
	if argCount != 1 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	n := dictStore.Exists(cmd.args[:1])
	if n > 0 {
		value := dictStore.Get(key)
		dictStore.Set(key, value, -1)
		return 1, nil
	}
	return 0, nil
}

// SADD key 1 2 3 4 5 -> 5
func HandleSetAdd(cmd *Command) (int, error) {
	argCount := len(cmd.args)
	if argCount < 1 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	existInDict := dictStore.Exists(cmd.args[:1])
	if existInDict > 0 {
		return 0, errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	set, exists := setStore[key]
	if !exists {
		set = data_structure.NewSet()
		setStore[key] = set
	}
	added := set.Add(cmd.args[1:]...)
	return added, nil
}

// SISMEMBER key valueExist -> 1
// SISMEMBER key valueNotExist -> 0
func HandleSISMEMBER(cmd *Command) (int, error) {
	argCount := len(cmd.args)
	if argCount != 2 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}

	key := cmd.args[0]
	value := cmd.args[1]

	set, exists := setStore[key]
	if exists && set.IsMember(value) {
		return 1, nil
	}
	return 0, nil
}

// SREM key value1 value2 -> 2
func HandleSREM(cmd *Command) (int, error) {
	argCount := len(cmd.args)
	if argCount < 1 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	set, exists := setStore[key]
	if !exists {
		return 0, nil
	}
	removed := set.Remove(cmd.args[1:]...)
	return removed, nil
}

// SMEMBERS key -> list member
func HandleSMEMBERS(cmd *Command) ([]string, error) {
	argCount := len(cmd.args)
	if argCount != 1 {
		return nil, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	set, exists := setStore[key]
	if !exists {
		return nil, nil
	}
	members := set.Members()
	return members, nil
}

// flush all
func HandleFlushDb(cmd *Command) (int, error) {
	InitStorage()
	return 1, nil
}

// ZADD myzset score(float64) member -> 1
func HandleZADD(cmd *Command) (int, error) {
	argCount := len(cmd.args)
	if argCount < 3 || (argCount-1)%2 != 0 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	skipList, exist := skipListStore[key]
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
	skipListStore[key] = skipList
	return added, nil
}

// ZSCORE zmyset member
func HandleZSCORE(cmd *Command) (any, error) {
	argCount := len(cmd.args)
	if argCount < 2 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	skipList, exist := skipListStore[key]
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
	skipList, exist := skipListStore[key]
	if !exist {
		return constant.NILL, nil
	}
	value := cmd.args[1]
	rank, err := skipList.GetRank(value)
	if err != nil || rank < 0 {
		return constant.NILL, err
	}
	return rank, nil
}

// ---------------- COUNT MIN SKETCH -----------------------

// CMS.INITBYPROB key error probability
func HandleINITBYPROB(cmd *Command) (string, error) {
	argCount := len(cmd.args)
	if argCount != 3 {
		return "", fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.args[0]
	_, exist := cmsStore[key]
	if exist {
		return "", fmt.Errorf("CMS: %s already exists", key)
	}

	errorCms, err := strconv.ParseFloat(cmd.args[1], 64)
	if err != nil || errorCms <= 0 || errorCms >= 1 {
		return "", fmt.Errorf("Invalid argument error value")
	}

	probCms, err := strconv.ParseFloat(cmd.args[2], 64)
	if err != nil || probCms <= 0 || probCms >= 1 {
		return "", fmt.Errorf("Invalid argument probability value")
	}

	cmsStore[key] = data_structure.NewCMS(errorCms, probCms)
	return "OK", nil
}

// CMS.INCRBY key item increment [item increment ...]
func HandleINCRBY(cmd *Command) ([]uint64, error) {
	argCount := len(cmd.args)
	if argCount < 3 || (argCount-1)%2 != 0 {
		return nil, fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.args[0]
	cms, exist := cmsStore[key]
	if !exist {
		return nil, fmt.Errorf("CMS %s doesn't exist", key)
	}

	counts := make([]uint64, (argCount-1)/2)
	for i := 1; i < argCount; i = i + 2 {
		item := cmd.args[i]
		increment, err := strconv.ParseUint(cmd.args[i+1], 10, 64)
		if err != nil {
			return nil, errors.New("value increment is not a valid float")
		}
		count := cms.IncreaseBy(item, increment)
		counts[i/2] = count
	}

	return counts, nil
}

// CMS.QUERY key item item1 item2
func HandleQUERY(cmd *Command) ([]uint64, error) {
	argCount := len(cmd.args)
	if argCount < 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.args[0]
	cms, exist := cmsStore[key]
	if !exist {
		return nil, fmt.Errorf("CMS %s doesn't exist", key)
	}

	counts := make([]uint64, argCount-1)
	for i := 1; i < argCount; i++ {
		count := cms.Query(cmd.args[i])
		counts[i-1] = count
	}

	return counts, nil
}
