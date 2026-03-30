package resp

import (
	"fmt"
	"testing"
)

func TestSimpleStringDecode(t *testing.T) {
	cases := map[string]string{
		"+OK\r\n": "OK",
	}
	for k, v := range cases {
		value, _ := Decode([]byte(k))
		if v != value {
			t.Fail()
		}
	}
}

func TestError(t *testing.T) {
	cases := map[string]string{
		"-Error message\r\n": "Error message",
	}
	for k, v := range cases {
		value, _ := Decode([]byte(k))
		if v != value {
			t.Fail()
		}
	}
}

func TestInt64(t *testing.T) {
	cases := map[string]int64{
		":0\r\n":    0,
		":1000\r\n": 1000,
	}
	for k, v := range cases {
		value, _ := Decode([]byte(k))
		if v != value {
			t.Fail()
		}
	}
}

func TestBulkStringDecode(t *testing.T) {
	cases := map[string]string{
		"$5\r\nhello\r\n":        "hello",
		"$0\r\n\r\n":             "",
		"$11\r\nhello world\r\n": "hello world",
	}
	for k, v := range cases {
		value, _ := Decode([]byte(k))
		if v != value {
			t.Fail()
		}
	}
}

func TestBulkString(t *testing.T) {
	type testCase struct {
		name    string
		input   string
		want    string
		wantErr bool
	}

	cases := []testCase{
		{
			name:  "normal string",
			input: "$5\r\nhello\r\n",
			want:  "hello",
		},
		{
			name:  "empty string",
			input: "$0\r\n\r\n",
			want:  "",
		},
		{
			name:  "long string",
			input: "$11\r\nhello world\r\n",
			want:  "hello world",
		},
		{
			name:  "null bulk string",
			input: "$-1\r\n",
			want:  "",
		},
		{
			name:    "missing CRLF",
			input:   "$5\r\nhello",
			wantErr: true,
		},
		{
			name:    "invalid length",
			input:   "$abc\r\nhello\r\n",
			wantErr: true,
		},
		{
			name:    "length larger than data",
			input:   "$10\r\nhello\r\n",
			wantErr: true,
		},
		{
			name:    "not bulk string",
			input:   "+OK\r\n",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := ReadBulkString([]byte(tc.input))

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error but got none, value=%q", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestArrayDecode(t *testing.T) {
	cases := map[string][]interface{}{
		"*0\r\n":                                                   {},
		"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n":                     {"hello", "world"},
		"*3\r\n:1\r\n:2\r\n:3\r\n":                                 {int64(1), int64(2), int64(3)},
		"*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$5\r\nhello\r\n":            {int64(1), int64(2), int64(3), int64(4), "hello"},
		"*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n": {[]int64{int64(1), int64(2), int64(3)}, []interface{}{"Hello", "World"}},
	}
	for k, v := range cases {
		value, _ := Decode([]byte(k))
		array := value.([]interface{})
		if len(array) != len(v) {
			t.Fail()
		}
		for i := range array {
			if fmt.Sprintf("%v", v[i]) != fmt.Sprintf("%v", array[i]) {
				t.Fail()
			}
		}
	}
}

//func TestEncodeString2DArray(t *testing.T) {
//	var decode = [][]string{{"hello", "world"}, {"1", "2", "3"}, {"xyz"}}
//	encode := Encode(decode, false)
//	assert.EqualValues(t, "*3\r\n*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n*3\r\n$1\r\n1\r\n$1\r\n2\r\n$1\r\n3\r\n*1\r\n$3\r\nxyz\r\n", string(encode))
//	decodeAgain, _ := Decode(encode)
//	for i := 0; i < 3; i++ {
//		for j := 0; j < len(decode[i]); j++ {
//			assert.EqualValues(t, decode[i][j], decodeAgain.([]interface{})[i].([]interface{})[j])
//		}
//	}
//}
