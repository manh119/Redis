package command

import (
	"errors"
	"fmt"
	"strings"
)

// example: set key value EX 100
// => cmd = set, args = ["key", "value", "EX", "100"]
type Command struct {
	Cmd  string
	Args []string
}

func NewCommand(cmd string, args []string) *Command {
	return &Command{Cmd: cmd, Args: args}
}

func HandleCommand(decodeRequest any) (any, error) {
	arr, ok := decodeRequest.([]any)
	if !ok || decodeRequest == nil || len(arr) == 0 {
		return "", errors.New("invalid command")
	}

	cmdName, cmd := convertToCommand(arr)

	switch cmdName {
	case "PING":
		return HandlePing(cmd)
	case "GET":
		return HandleGet(cmd)
	case "SET":
		return HandleSet(cmd)
	case "TTL":
		return HandleTTL(cmd)
	case "EXPIRE":
		return HandleExpire(cmd)
	case "DEL":
		return HandleDel(cmd)
	case "EXISTS":
		return HandleExists(cmd)
	case "SADD":
		return HandleSetAdd(cmd)
	case "SISMEMBER":
		return HandleSISMEMBER(cmd)
	case "SREM":
		return HandleSREM(cmd)
	case "SMEMBERS":
		return HandleSMEMBERS(cmd)
	case "FLUSHDB":
		return HandleFlushDb(cmd)
	case "PERSIST":
		return HandlePERSIST(cmd)
	case "ZADD":
		return HandleZADD(cmd)
	case "ZSCORE":
		return HandleZSCORE(cmd)
	case "ZRANK":
		return HandleZRANK(cmd)
	case "CMS.INITBYPROB":
		return HandleINITBYPROB(cmd)
	case "CMS.INCRBY":
		return HandleINCRBY(cmd)
	case "CMS.QUERY":
		return HandleQUERY(cmd)
	case "BF.RESERVE":
		return HandleRESERVE(cmd)
	case "BF.MADD":
		return HandleMADD(cmd)
	case "BF.MEXISTS":
		return HandleMEXISTS(cmd)
	default:
		return nil, errors.New("command is not supported")

	}
}

func convertToCommand(arr []any) (string, *Command) {
	cmdName := arr[0].(string)
	cmdName = strings.ToUpper(cmdName)
	var args []string
	for i := 1; i < len(arr); i++ {
		args = append(args, fmt.Sprintf("%v", arr[i]))
	}

	cmd := NewCommand(cmdName, args)
	return cmdName, cmd
}
