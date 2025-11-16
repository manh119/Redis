package resp

import (
	"bytes"
	"errors"
	"fmt"
)

const CRLF string = "\r\n"

var RespNil = []byte("$-1\r\n")

func Encode(value interface{}, isSimpleString bool) []byte {
	switch v := value.(type) {
	case string:
		if isSimpleString {
			return []byte(fmt.Sprintf("+%s%s", v, CRLF))
		}
		return []byte(fmt.Sprintf("$%d%s%s%s", len(v), CRLF, v, CRLF))
	case int64, int32, int16, int8, int:
		return []byte(fmt.Sprintf(":%d\r\n", v))
	case error:
		return []byte(fmt.Sprintf("-%s\r\n", v))
	case []string:
		return encodeStringArray(value.([]string))
	case [][]string:
		var b []byte
		buf := bytes.NewBuffer(b)
		for _, sa := range value.([][]string) {
			buf.Write(encodeStringArray(sa))
		}
		return []byte(fmt.Sprintf("*%d\r\n%s", len(value.([][]string)), buf.Bytes()))
	case []interface{}:
		var b []byte
		buf := bytes.NewBuffer(b)
		for _, x := range value.([]interface{}) {
			buf.Write(Encode(x, false))
		}
		return []byte(fmt.Sprintf("*%d\r\n%s", len(value.([]interface{})), buf.Bytes()))
	default:
		return RespNil
	}
}

func DecodeOne(data []byte) (interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, errors.New("no data")
	}
	switch data[0] {
	case '+':
		return readSimpleString(data)
	case ':':
		return readInt64(data)
	case '-':
		return readError(data)
	case '$':
		return readBulkString(data)
	case '*':
		return readArray(data)
	}
	return nil, 0, nil
}

func Decode(data []byte) (interface{}, error) {
	res, _, err := DecodeOne(data)
	return res, err
}

// +OK\r\n => OK, 5
func readSimpleString(data []byte) (string, int, error) {
	// Kiểm tra dữ liệu rỗng
	if len(data) == 0 || data[0] != '+' {
		return "", 0, errors.New("not a simple string")
	}

	// Tìm vị trí CRLF
	index := bytes.Index(data, []byte(CRLF))
	if index < 0 {
		return "", 0, errors.New("CRLF not found")
	}

	// Lấy substring giữa '+' và CRLF
	s := string(data[1:index])
	pos := index + len(CRLF)
	return s, pos, nil
}

// :123\r\n => 123
func readInt64(data []byte) (int64, int, error) {
	// Kiểm tra dữ liệu rỗng
	if len(data) == 0 || data[0] != ':' {
		return 0, 0, errors.New("not an integer")
	}

	// Tìm vị trí CRLF
	index := bytes.Index(data, []byte(CRLF))
	if index < 0 {
		return 0, 0, errors.New("CRLF not found")
	}

	// Lấy substring giữa ':' và CRLF
	var num int64
	_, err := fmt.Sscanf(string(data[1:index]), "%d", &num)
	if err != nil {
		return 0, 0, err
	}

	pos := index + len(CRLF)
	return num, pos, nil
}

func readError(data []byte) (string, int, error) {
	// Kiểm tra dữ liệu rỗng hoặc không phải error
	if len(data) == 0 || data[0] != '-' {
		return "", 0, errors.New("not an error string")
	}

	// Tìm vị trí CRLF
	index := bytes.Index(data, []byte(CRLF))
	if index < 0 {
		return "", 0, errors.New("CRLF not found")
	}

	// Lấy substring giữa '-' và CRLF
	s := string(data[1:index])
	pos := index + len(CRLF)
	return s, pos, nil
}

// $5\r\nhello\r\n => 5, 4
func readLen(data []byte) (int, int) {
	if len(data) == 0 {
		return 0, 0
	}
	index := bytes.Index(data, []byte(CRLF))
	if index < 0 {
		return 0, 0
	}
	var length int
	_, err := fmt.Sscanf(string(data[1:index]), "%d", &length)
	if err != nil {
		return 0, 0
	}
	pos := index + len(CRLF)
	return length, pos
}

// $5\r\nhello\r\n => "hello"
func readBulkString(data []byte) (string, int, error) {
	if len(data) == 0 || data[0] != '$' {
		return "", 0, errors.New("not a bulk string")
	}
	length, pos := readLen(data)
	if length == -1 {
		return "", pos, nil // Null bulk string
	}
	if pos+length+len(CRLF) > len(data) {
		return "", pos, errors.New("unexpected end of data")
	}
	s := string(data[pos : pos+length])
	pos += length + len(CRLF)
	return s, pos, nil
}

// *2\r\n$5\r\nhello\r\n$5\r\nworld\r\n => {"hello", "world"}
func readArray(data []byte) (interface{}, int, error) {
	length, pos := readLen(data)
	if length == -1 {
		return nil, pos, nil
	}
	res := make([]interface{}, length)
	for i := 0; i < length; i++ {
		if pos >= len(data) {
			return nil, pos, errors.New("unexpected end of data")
		}
		elem, n, err := DecodeOne(data[pos:])
		if err != nil {
			return nil, pos, err
		}
		res[i] = elem
		pos += n
	}
	return res, pos, nil
}

func encodeString(s string) []byte {
	return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(s), s))
}

func encodeStringArray(sa []string) []byte {
	var b []byte
	buf := bytes.NewBuffer(b)
	for _, s := range sa {
		buf.Write(encodeString(s))
	}
	return []byte(fmt.Sprintf("*%d\r\n%s", len(sa), buf.Bytes()))
}
