package core

import (
	"errors"
	"strconv"
)

type Command struct {
	Cmd  string
	Args []any
}

func HandleTTL(cmd Command) (any, error) {
	if len(cmd.Args) == 1 {
		key, ok := cmd.Args[0].(string)
		if !ok {
			return "", errors.New("ERR value is not a valid string")
		}
		return dictStore.Ttl(key), nil
	}
	return "", errors.New("invalid command")
}

func HandlePing(cmd Command) (string, error) {
	if cmd.Args == nil || len(cmd.Args) == 0 {
		return "PONG", nil
	}

	if len(cmd.Args) == 1 {
		return cmd.Args[0].(string), nil
	}

	return "", errors.New("invalid command")
}

func HandleGet(cmd Command) (any, error) {
	if len(cmd.Args) == 1 {
		key, ok := cmd.Args[0].(string)
		if !ok {
			return "", errors.New("ERR value is not a valid string")
		}
		return dictStore.Get(key), nil
	}
	return "", errors.New("invalid command")
}

// expire key 100s
func HandleExpire(cmd Command) (any, error) {
	if len(cmd.Args) == 2 {
		key, ok := cmd.Args[0].(string)
		if !ok {
			return "", errors.New("ERR value is not a valid string")
		}
		ttlStr, ok := cmd.Args[1].(string)
		if !ok {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		parsedTTL, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		return dictStore.Expire(key, parsedTTL*1000), nil
	}
	return "", errors.New("invalid number of args")
}

// exists key1 key2 -> 2
func HandleExists(cmd Command) (any, error) {
	if len(cmd.Args) == 0 {
		return "", errors.New("invalid number of args")
	}
	return dictStore.Exists(cmd.Args), nil
}

// del key1 key2
func HandleDel(cmd Command) (any, error) {
	if len(cmd.Args) == 0 {
		return "", errors.New("invalid number of args")
	}
	return dictStore.Del(cmd.Args), nil
}

func HandleSet(cmd Command) (string, error) {
	argCount := len(cmd.Args)
	if argCount != 2 && argCount != 4 {
		return "", errors.New("ERR wrong number of arguments for 'set' command")
	}
	key, ok1 := cmd.Args[0].(string)
	if !ok1 {
		return "", errors.New("ERR key must be a string")
	}
	value := cmd.Args[1]
	var ttl int64 = -1
	if argCount == 4 {
		ttlStr, ok := cmd.Args[3].(string)
		if !ok {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		parsedTTL, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		ttl = parsedTTL
	}
	dictStore.Set(key, value, ttl*1000)
	return "OK", nil
}
