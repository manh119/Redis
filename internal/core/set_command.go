package core

import (
	"errors"
	"fmt"

	"github.com/manh119/Redis/internal/core/data_structure"
)

// SADD key 1 2 3 4 5 -> 5
func HandleSetAdd(cmd *Command) (int, error) {
	argCount := len(cmd.args)
	if argCount < 1 {
		return 0, errors.New(fmt.Sprintf("ERR wrong number of arguments for '%s' command", cmd.Cmd))
	}
	key := cmd.args[0]
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
	_, exists := setStore[key]
	if exists {
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
	return set.Members(), nil
}

// flush all
func HandleFlushDb(cmd *Command) (int, error) {
	InitStorage()
	return 1, nil
}
