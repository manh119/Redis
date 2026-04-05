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
	if len(cmd.args) == 1 {
		key := cmd.args[0]
		return core.DictStore.Ttl(key), nil
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
		return core.DictStore.Get(key), nil
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
		return core.DictStore.Expire(key, parsedTTL*1000), nil
	}
	return "", errors.New("invalid number of args")
}

// exists key1 key2 -> 2
func HandleExists(cmd *Command) (any, error) {
	if len(cmd.args) == 0 {
		return "", errors.New("invalid number of args")
	}
	return core.DictStore.Exists(cmd.args), nil
}

// del key1 key2
func HandleDel(cmd *Command) (any, error) {
	if len(cmd.args) == 0 {
		return "", errors.New("invalid number of args")
	}
	return core.DictStore.Del(cmd.args), nil
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
			core.DictStore.Set(key, value, ttl*1000)
		} else if strings.ToUpper(cmd.args[2]) == "PX" { // ttl in miliSecond
			core.DictStore.Set(key, value, ttl)
		} else {
			return "", errors.New("ERR unknown command")
		}
	}
	core.DictStore.Set(key, value, ttl)
	return "OK", nil
}

// PERSIST key -> set ttl = -1 if key is valid
func HandlePERSIST(cmd *Command) (int, error) {
	argCount := len(cmd.args)
	if argCount != 1 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
	n := core.DictStore.Exists(cmd.args[:1])
	if n > 0 {
		value := core.DictStore.Get(key)
		core.DictStore.Set(key, value, -1)
		return 1, nil
	}
	return 0, nil
}

// flush all
func HandleFlushDb(cmd *Command) (int, error) {
	core.InitStorage()
	return 1, nil
}
