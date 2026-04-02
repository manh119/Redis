package core

import (
	"errors"
	"strconv"
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
	var ttlInSecond int64 = -1
	if argCount == 4 {
		ttlStr := cmd.args[3]
		parsedTTL, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil {
			return "", errors.New("ERR value is not an integer or out of range")
		}
		ttlInSecond = parsedTTL
	}
	dictStore.Set(key, value, ttlInSecond*1000)
	return "OK", nil
}
