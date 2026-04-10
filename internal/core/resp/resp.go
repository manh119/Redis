package resp

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/manh119/Redis/internal/config"
)

const CRLF string = "\r\n"

func DecodeOne(data []byte) (any, int, error) {
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
		return ReadBulkString(data)
	case '*':
		return ReadArray(data)
	}
	return nil, 0, nil
}

func Decode(data []byte) (any, error) {
	res, _, err := DecodeOne(data)
	return res, err
}

// +OK\r\n => OK, 5 (next index)
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

// $5\r\nhello\r\n => "hello"
// $-1\r\n => ""
// $0\r\n\r\n => ""
func ReadBulkString(data []byte) (string, int, error) {
	// 1. Kiểm tra định dạng cơ bản
	if len(data) < 4 || data[0] != '$' {
		return "", 0, errors.New("không phải định dạng Bulk String (thiếu '$')")
	}

	// 2. Tìm vị trí kết thúc của dòng tiêu đề (ví dụ: tìm vị trí sau số 5 trong "$5\r\n")
	headerEnd := bytes.Index(data, []byte(CRLF))
	if headerEnd == -1 {
		return "", 0, errors.New("thiếu ký tự xuống dòng sau phần độ dài")
	}

	// 3. Lấy giá trị độ dài (nằm giữa '$' và '\r\n')
	lenRaw := string(data[1:headerEnd])
	length, err := strconv.Atoi(lenRaw)
	if err != nil {
		return "", 0, errors.New("độ dài không hợp lệ")
	}

	// Trường hợp đặc biệt: Null Bulk String ($-1\r\n)
	if length == -1 {
		return "", headerEnd + len(CRLF), nil
	}

	// 4. Xác định vị trí dữ liệu thực tế
	bodyStart := headerEnd + len(CRLF)
	bodyEnd := bodyStart + length
	totalExpectedLen := bodyEnd + len(CRLF)

	// 5. Kiểm tra xem toàn bộ gói tin có đủ độ dài không
	if len(data) < totalExpectedLen {
		return "", 0, errors.New("dữ liệu thực tế ngắn hơn độ dài khai báo")
	}

	// 6. Kiểm tra xem có kết thúc bằng \r\n không
	if !bytes.Equal(data[bodyEnd:totalExpectedLen], []byte(CRLF)) {
		return "", 0, errors.New("thiếu ký tự kết thúc \r\n ở cuối chuỗi")
	}

	// 7. Trả về kết quả
	result := string(data[bodyStart:bodyEnd])
	return result, totalExpectedLen, nil
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

// *2\r\n$5\r\nhello\r\n$5\r\nworld\r\n => {"hello", "world"}
func ReadArray(data []byte) (any, int, error) {
	if data == nil || len(data) == 0 || data[0] != '*' {
		return nil, 0, errors.New("not an array")
	}
	endHeader := bytes.Index(data, []byte(CRLF))
	if endHeader == -1 {
		return nil, 0, errors.New("CRLF not found")
	}
	length, err := strconv.Atoi(string(data[1:endHeader]))
	if err != nil {
		return nil, 0, errors.New("fail to convert length of array")
	}
	if length == -1 {
		return nil, 0, nil
	}

	// start body
	pos := endHeader + len(CRLF)
	res := make([]any, length)
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

func Encode(response any) (string, error) {
	if response == nil {
		return config.NILL, nil
	}

	// handle slice, array
	val := reflect.ValueOf(response)
	kind := val.Kind()

	if kind == reflect.Slice || kind == reflect.Array {
		// Exceptional: []byte -> Bulk String ($), not Array (*)
		if kind == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8 {
			bytes := response.([]byte)
			return fmt.Sprintf("$%d\r\n%s\r\n", len(bytes), string(bytes)), nil
		}

		// handle Array (*)
		var sb strings.Builder
		n := val.Len()
		sb.WriteString(fmt.Sprintf("*%d\r\n", n))

		for i := 0; i < n; i++ {
			encodedEle, err := Encode(val.Index(i).Interface())
			if err != nil {
				return "", err
			}
			sb.WriteString(encodedEle)
		}
		return sb.String(), nil
	}

	// other case
	switch value := response.(type) {
	case nil:
		return config.NILL, nil
	case string: // simple string
		return fmt.Sprintf("+%s\r\n", value), nil
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return fmt.Sprintf(":%d\r\n", value), nil
	case float64, float32:
		valStr := fmt.Sprintf("%g", value)
		return fmt.Sprintf("$%d\r\n%s\r\n", len(valStr), valStr), nil
	case error:
		return fmt.Sprintf("-%s\r\n", value), nil
	default:
		return "", errors.New(fmt.Sprintf("type %s is not supported", value))
	}
}
