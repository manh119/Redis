package command

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/manh119/Redis/internal/core"
	"github.com/manh119/Redis/internal/core/data_structure"
)

// ---------------- COUNT MIN SKETCH -----------------------

// CMS.INITBYPROB key error probability
func HandleINITBYPROB(cmd *Command) (string, error) {
	argCount := len(cmd.Args)
	if argCount != 3 {
		return "", fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.Args[0]
	_, exist := storage.CmsStore[key]
	if exist {
		return "", fmt.Errorf("CMS: %s already exists", key)
	}

	errorCms, err := strconv.ParseFloat(cmd.Args[1], 64)
	if err != nil || errorCms <= 0 || errorCms >= 1 {
		return "", fmt.Errorf("Invalid argument error value")
	}

	probCms, err := strconv.ParseFloat(cmd.Args[2], 64)
	if err != nil || probCms <= 0 || probCms >= 1 {
		return "", fmt.Errorf("Invalid argument probability value")
	}

	storage.CmsStore[key] = data_structure.NewCMS(errorCms, probCms)
	return "OK", nil
}

// CMS.INCRBY key item increment [item increment ...]
func HandleINCRBY(cmd *Command) ([]uint64, error) {
	argCount := len(cmd.Args)
	if argCount < 3 || (argCount-1)%2 != 0 {
		return nil, fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.Args[0]
	cms, exist := storage.CmsStore[key]
	if !exist {
		return nil, fmt.Errorf("CMS %s doesn't exist", key)
	}

	counts := make([]uint64, (argCount-1)/2)
	for i := 1; i < argCount; i = i + 2 {
		item := cmd.Args[i]
		increment, err := strconv.ParseUint(cmd.Args[i+1], 10, 64)
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
	argCount := len(cmd.Args)
	if argCount < 2 {
		return nil, fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd.Cmd)
	}

	key := cmd.Args[0]
	cms, exist := storage.CmsStore[key]
	if !exist {
		return nil, fmt.Errorf("CMS %s doesn't exist", key)
	}

	counts := make([]uint64, argCount-1)
	for i := 1; i < argCount; i++ {
		count := cms.Query(cmd.Args[i])
		counts[i-1] = count
	}

	return counts, nil
}
