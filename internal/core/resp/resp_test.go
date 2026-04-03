package resp

import (
	"errors"
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
	cases := map[string][]any{
		"*0\r\n":                                                   {},
		"*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n":                     {"hello", "world"},
		"*3\r\n:1\r\n:2\r\n:3\r\n":                                 {int64(1), int64(2), int64(3)},
		"*5\r\n:1\r\n:2\r\n:3\r\n:4\r\n$5\r\nhello\r\n":            {int64(1), int64(2), int64(3), int64(4), "hello"},
		"*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Hello\r\n-World\r\n": {[]int64{int64(1), int64(2), int64(3)}, []any{"Hello", "World"}},
	}
	for k, v := range cases {
		value, _ := Decode([]byte(k))
		array := value.([]any)
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

func TestEncode(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
		wantErr  bool
	}{
		{
			name:     "Simple String",
			input:    "OK",
			expected: "+OK\r\n",
		},
		{
			name:     "Error",
			input:    errors.New("Error Message"),
			expected: "-Error Message\r\n",
		},
		{
			name:     "Integer",
			input:    1024,
			expected: ":1024\r\n",
		},
		{
			name:     "Bulk String",
			input:    []byte("hello"),
			expected: "$5\r\nhello\r\n",
		},
		{
			name:     "Empty Bulk String",
			input:    []byte(""),
			expected: "$0\r\n\r\n",
		},
		{
			name:     "Array of Mixed Types",
			input:    []any{"PING", 100},
			expected: "*2\r\n+PING\r\n:100\r\n",
		},
		{
			name:     "Array of String",
			input:    []string{"PING", "100"},
			expected: "*2\r\n+PING\r\n+100\r\n",
		},
		{
			name:     "Nested Array",
			input:    []any{[]any{"echo", "hi"}},
			expected: "*1\r\n*2\r\n+echo\r\n+hi\r\n",
		},
		{
			name:    "Unsupported Type",
			input:   struct{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("Encode() = %q, want %q", got, tt.expected)
			}
		})
	}
}
