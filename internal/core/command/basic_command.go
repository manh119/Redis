package command

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/manh119/Redis/internal/core"
)

// -------------------- Basic command -----------------------
func HandleTTL(cmd *Command) (any, error) {
	if len(cmd.Args) == 1 {
		key := cmd.Args[0]
		return storage.DictStore.Ttl(key), nil
	}
	return "", errors.New("invalid command")
}

func HandlePing(cmd *Command) (string, error) {
	if cmd.Args == nil || len(cmd.Args) == 0 {
		return "PONG", nil
	}

	if len(cmd.Args) == 1 {
		return cmd.Args[0], nil
	}

	return "", errors.New("invalid command")
}

func HandleGet(cmd *Command) (string, error) {
	if len(cmd.Args) == 1 {
		key := cmd.Args[0]
		value, err := storage.DictStore.Get(key)
		if err != nil {
			return "", err
		}
		return value, nil
	}
	return "", errors.New("invalid command")
}

// expire key 100s
func HandleExpire(cmd *Command) (any, error) {
	if len(cmd.Args) == 2 {
		key := cmd.Args[0]
		ttl := cmd.Args[1]
		parsedTTL, err := strconv.ParseInt(ttl, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		return storage.DictStore.Expire(key, parsedTTL*1000), nil
	}
	return "", errors.New("invalid number of args")
}

// exists key1 key2 -> 2
func HandleExists(cmd *Command) (any, error) {
	if len(cmd.Args) == 0 {
		return "", errors.New("invalid number of args")
	}
	return storage.DictStore.Exists(cmd.Args), nil
}

// del key1 key2
func HandleDel(cmd *Command) (any, error) {
	if len(cmd.Args) == 0 {
		return "", errors.New("invalid number of args")
	}
	return storage.DictStore.Del(cmd.Args), nil
}

// set key value
// set key value EX ttl
func HandleSet(cmd *Command) (string, error) {
	argCount := len(cmd.Args)
	if argCount != 2 && argCount != 4 {
		return "", errors.New("ERR wrong number of arguments for 'set' command")
	}
	key := cmd.Args[0]
	value := cmd.Args[1]
	var ttl int64 = -1
	if argCount == 4 {
		ttlStr := cmd.Args[3]
		parsedTTL, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		ttl = parsedTTL
		if strings.ToUpper(cmd.Args[2]) == "EX" {
			storage.DictStore.Set(key, value, ttl*1000)
		} else if strings.ToUpper(cmd.Args[2]) == "PX" { // ttl in miliSecond
			storage.DictStore.Set(key, value, ttl)
		} else {
			return "", errors.New("ERR unknown command")
		}
	}
	storage.DictStore.Set(key, value, ttl)
	return "OK", nil
}

// PERSIST key -> set ttl = -1 if key is valid
func HandlePERSIST(cmd *Command) (int, error) {
	argCount := len(cmd.Args)
	if argCount != 1 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.Args[0]
	n := storage.DictStore.Exists(cmd.Args[:1])
	if n > 0 {
		value, err := storage.DictStore.Get(key)
		if err != nil {
			return 0, err
		}
		storage.DictStore.Set(key, value, -1)
		return 1, nil
	}
	return 0, nil
}

// flush all
func HandleFlushDb(cmd *Command) (int, error) {
	storage.InitStorage()
	return 1, nil
}
