// miniredis_commented.go
// Minimal Redis-like server in Go using epoll (Linux).
// Phiên bản này chứa comment rất chi tiết theo yêu cầu: mỗi bước trong hàm
// được ghi chú theo kiểu số thứ tự (1., 2., ...).

package main

import (
	"container/heap"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ---------- Data structures (in-memory store) ----------

type ValueType int

const (
	StringType ValueType = iota
	SortedSetType
)

// Value: lưu trữ 1 key có thể là string hoặc sorted set, kèm expire
type Value struct {
	// 1. typ: kiểu value (string hoặc zset)
	typ ValueType
	// 2. strVal: giá trị khi là string
	strVal string

	// 3. zset: con trỏ đến sorted set khi typ==SortedSetType
	zset *SortedSet

	// 4. expireAt: thời điểm hết hạn; zero value nghĩa là không có TTL
	expireAt time.Time
}

// Store: map + mutex để bảo vệ concurrent access
type Store struct {
	mu sync.RWMutex
	m  map[string]*Value
}

func NewStore() *Store {
	// 1. tạo struct store
	// 2. khởi tạo map
	return &Store{m: make(map[string]*Value)}
}

// setString: SET key value [EX seconds]
func (s *Store) setString(key, val string, exSeconds int) {
	// 1. lock để đảm bảo ghi an toàn
	s.mu.Lock()
	defer s.mu.Unlock()
	// 2. tạo Value mới với kiểu string
	v := &Value{typ: StringType, strVal: val}
	// 3. nếu có ttl, set expireAt
	if exSeconds > 0 {
		v.expireAt = time.Now().Add(time.Duration(exSeconds) * time.Second)
	}
	// 4. gán vào map (ghi/overwrite)
	s.m[key] = v
}

// getString: lấy value nếu tồn tại và chưa expired
func (s *Store) getString(key string) (string, bool) {
	// 1. RLock để đọc
	s.mu.RLock()
	defer s.mu.RUnlock()
	// 2. lấy value từ map
	v, ok := s.m[key]
	if !ok || v.typ != StringType {
		// 3. nếu không tồn tại hoặc không phải string -> trả false
		return "", false
	}
	// 4. nếu có expire và đã quá hạn -> xoá key
	if !v.expireAt.IsZero() && time.Now().After(v.expireAt) {
		// 4.1: Lưu ý: ta cần unlock RLock trước khi Lock để xóa
		s.mu.RUnlock()
		s.mu.Lock()
		delete(s.m, key)
		s.mu.Unlock()
		s.mu.RLock()
		return "", false
	}
	// 5. trả về giá trị
	return v.strVal, true
}

// ttl: trả về thời gian còn lại theo giây
func (s *Store) ttl(key string) int {
	// 1. RLock
	s.mu.RLock()
	defer s.mu.RUnlock()
	// 2. lấy value
	v, ok := s.m[key]
	if !ok {
		// 3. -2 theo Redis khi key không tồn tại
		return -2
	}
	// 4. nếu không có expire -> -1
	if v.expireAt.IsZero() {
		return -1
	}
	// 5. tính remaining seconds
	remaining := int(time.Until(v.expireAt).Seconds())
	if remaining < 0 {
		return -2
	}
	return remaining
}

// ensureZSet: tạo mới nếu chưa có hoặc không phải zset
func (s *Store) ensureZSet(key string) *SortedSet {
	// 1. lock vì có thể ghi
	s.mu.Lock()
	defer s.mu.Unlock()
	// 2. kiểm tra key
	v, ok := s.m[key]
	if !ok || v.typ != SortedSetType {
		// 3. tạo và gán
		z := NewSortedSet()
		v = &Value{typ: SortedSetType, zset: z}
		s.m[key] = v
		return z
	}
	// 4. trả về zset hiện có
	return v.zset
}

// getZSet: lấy zset nếu tồn tại và đúng kiểu
func (s *Store) getZSet(key string) (*SortedSet, bool) {
	// 1. RLock
	s.mu.RLock()
	defer s.mu.RUnlock()
	// 2. lấy value
	v, ok := s.m[key]
	if !ok || v.typ != SortedSetType {
		return nil, false
	}
	return v.zset, true
}

// ---------- Sorted set (simple implementation) ----------

type ZItem struct {
	member string
	score  float64
}

// SortedSet: map để tra member nhanh + slice để giữ thứ tự
type SortedSet struct {
	members map[string]float64
	arr     []*ZItem // kept sorted by score ascending; small-scale implementation
}

func NewSortedSet() *SortedSet {
	// 1. tạo map và slice
	return &SortedSet{
		members: make(map[string]float64),
		arr:     make([]*ZItem, 0),
	}
}

// Add: thêm hoặc cập nhật member
func (z *SortedSet) Add(score float64, member string) {
	// 1. nếu member tồn tại -> update score
	if _, ok := z.members[member]; ok {
		z.members[member] = score
		// 2. cập nhật trong slice
		for _, it := range z.arr {
			if it.member == member {
				it.score = score
				break
			}
		}
		// 3. sắp xếp lại slice
		z.reSort()
		return
	}
	// 4. nếu member mới: ghi map và append slice
	z.members[member] = score
	z.arr = append(z.arr, &ZItem{member: member, score: score})
	// 5. sắp xếp lại
	z.reSort()
}

// reSort: sắp xếp lại arr theo score tăng dần (sử dụng heap làm ví dụ)
func (z *SortedSet) reSort() {
	// 1. tạo heap wrapper
	h := &zHeap{items: z.arr}
	// 2. khởi tạo heap (heap.Init sẽ heapify)
	heap.Init(h)
	// 3. pop tất cả để tạo slice mới đã sort
	out := make([]*ZItem, 0, len(z.arr))
	for h.Len() > 0 {
		out = append(out, heap.Pop(h).(*ZItem))
	}
	// 4. gán lại arr
	z.arr = out
}

// Range: lấy slice từ start..stop (hỗ trợ negative index)
func (z *SortedSet) Range(start, stop int) []ZItem {
	// 1. n = số phần tử
	n := len(z.arr)
	if n == 0 {
		return nil
	}
	// 2. xử lý chỉ số âm
	if start < 0 {
		start = n + start
	}
	if stop < 0 {
		stop = n + stop
	}
	// 3. clamp
	if start < 0 {
		start = 0
	}
	if stop < 0 {
		return nil
	}
	if start > stop || start >= n {
		return nil
	}
	if stop >= n {
		stop = n - 1
	}
	// 4. thu kết quả
	out := make([]ZItem, 0, stop-start+1)
	for i := start; i <= stop; i++ {
		out = append(out, *z.arr[i])
	}
	return out
}

// zHeap implements heap.Interface để sắp xếp ZItem theo score
type zHeap struct {
	items []*ZItem
}

func (h zHeap) Len() int { return len(h.items) }
func (h zHeap) Less(i, j int) bool {
	return h.items[i].score < h.items[j].score || (h.items[i].score == h.items[j].score && h.items[i].member < h.items[j].member)
}
func (h zHeap) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *zHeap) Push(x interface{}) {
	// 1. append item
	h.items = append(h.items, x.(*ZItem))
}
func (h *zHeap) Pop() interface{} {
	// 1. pop last
	n := len(h.items)
	x := h.items[n-1]
	h.items = h.items[:n-1]
	return x
}

// ---------- RESP parser (minimal) ----------
// Lưu ý: parser này hỗ trợ cả inline và array-bulk form cơ bản.

type ConnBuf struct {
	fd   int
	buf  []byte
	head int
	tail int
}

func NewConnBuf() *ConnBuf {
	// 1. tạo buffer với capacity ban đầu
	return &ConnBuf{buf: make([]byte, 0, 4096)}
}

func (cb *ConnBuf) appendData(p []byte) {
	// 1. đơn giản append bytes vào buf
	cb.buf = append(cb.buf, p...)
}

// parseRESPArray: parse 1 request (một RESP array hoặc inline) từ data
// trả về: args, số byte đã consume, error ("incomplete" nếu cần thêm dữ liệu)
func parseRESPArray(data []byte) ([]string, int, error) {
	// 1. nếu rỗng -> incomplete
	if len(data) == 0 {
		return nil, 0, errors.New("empty")
	}
	// 2. nếu không bắt đầu bằng '*' -> coi là inline command
	if data[0] != '*' {
		// 2.1: tìm CRLF
		lineEnd := bytesIndexCRLF(data)
		if lineEnd < 0 {
			return nil, 0, errors.New("incomplete")
		}
		// 2.2: tách theo whitespace
		line := strings.TrimSpace(string(data[:lineEnd]))
		parts := splitBySpacePreserve(line)
		// 2.3: trả về used = lineEnd + 2 (bỏ CRLF)
		return parts, lineEnd + 2, nil
	}
	// 3. parse array length: sau ký tự '*'
	pos := 1
	newline := indexOfCRLFFrom(data, pos)
	if newline < 0 {
		return nil, 0, errors.New("incomplete")
	}
	numStr := string(data[pos:newline])
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return nil, 0, err
	}
	pos = newline + 2
	out := make([]string, 0, num)
	// 4. cho mỗi phần tử: phải là $<len>\r\n<data>\r\n
	for i := 0; i < num; i++ {
		if pos >= len(data) {
			return nil, 0, errors.New("incomplete")
		}
		if data[pos] != '$' {
			return nil, 0, errors.New("protocol error: expected $")
		}
		pos++
		nl := indexOfCRLFFrom(data, pos)
		if nl < 0 {
			return nil, 0, errors.New("incomplete")
		}
		lenStr := string(data[pos:nl])
		bulkLen, err := strconv.Atoi(lenStr)
		if err != nil {
			return nil, 0, err
		}
		pos = nl + 2
		if pos+bulkLen+2 > len(data) {
			// 4.1: nếu chưa đủ bytes -> incomplete
			return nil, 0, errors.New("incomplete")
		}
		// 4.2: đọc bulk string
		part := string(data[pos : pos+bulkLen])
		out = append(out, part)
		pos = pos + bulkLen
		// 4.3: kiểm tra CRLF kết thúc bulk
		if !(pos+1 < len(data) && data[pos] == '\r' && data[pos+1] == '\n') {
			return nil, 0, errors.New("protocol error: missing CRLF after bulk")
		}
		pos += 2
	}
	// 5. trả về args và số byte đã dùng
	return out, pos, nil
}

// helper: tìm CRLF
func bytesIndexCRLF(b []byte) int {
	for i := 0; i+1 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' {
			return i
		}
	}
	return -1
}
func indexOfCRLFFrom(b []byte, start int) int {
	for i := start; i+1 < len(b); i++ {
		if b[i] == '\r' && b[i+1] == '\n' {
			return i
		}
	}
	return -1
}

func splitBySpacePreserve(s string) []string {
	f := strings.Fields(s)
	return f
}

// ---------- RESP builders (helpers để trả client) ----------
func respSimpleString(s string) []byte {
	// 1. +<string>\r\n
	return []byte("+" + s + "\r\n")
}
func respError(s string) []byte {
	// 1. -<error>\r\n
	return []byte("-" + s + "\r\n")
}
func respBulkString(s string) []byte {
	// 1. $<len>\r\n<string>\r\n
	if s == "" {
		return []byte("$0\r\n\r\n")
	}
	return []byte("$" + strconv.Itoa(len(s)) + "\r\n" + s + "\r\n")
}
func respNullBulk() []byte {
	// 1. $-1\r\n
	return []byte("$-1\r\n")
}
func respInteger(i int) []byte {
	// 1. :<int>\r\n
	return []byte(":" + strconv.Itoa(i) + "\r\n")
}
func respArrayStrings(arr []string) []byte {
	// 1. build array response
	if arr == nil {
		return []byte("*-1\r\n")
	}
	sb := strings.Builder{}
	sb.WriteString("*" + strconv.Itoa(len(arr)) + "\r\n")
	for _, s := range arr {
		sb.WriteString("$" + strconv.Itoa(len(s)) + "\r\n")
		sb.WriteString(s + "\r\n")
	}
	return []byte(sb.String())
}

// ---------- Epoll server ----------
const (
	MaxEvents = 128
)

type EpollServer struct {
	fd        int // epoll fd
	listener  net.Listener
	store     *Store
	conns     map[int]*ConnBuf
	connsLock sync.Mutex
}

func NewEpollServer() *EpollServer {
	// 1. tạo server, khởi tạo store và map
	return &EpollServer{
		store: NewStore(),
		conns: make(map[int]*ConnBuf),
	}
}

// Start: bind, set non-blocking, thêm listener vào epoll, vòng event loop
func (epollServer *EpollServer) Start(addr string) error {
	// 1️⃣ Tạo listener TCP trên địa chỉ addr
	log.Printf("listening on %s\n", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	epollServer.listener = listener

	// 2️⃣ Lấy file descriptor của listener để dùng syscall
	file, err := listener.(*net.TCPListener).File()
	if err != nil {
		return err
	}
	defer file.Close()
	sockFd := int(file.Fd()) // listener fd

	// 3️⃣ Tạo epoll instance
	epfd, err := syscall.EpollCreate1(0)
	if err != nil {
		return err
	}
	epollServer.fd = epfd

	// 4️⃣ Đặt listener socket thành non-blocking
	// => accept() sẽ không block, chỉ trả về ngay khi có client chờ
	if err := syscall.SetNonblock(sockFd, true); err != nil {
		return err
	}

	// 5️⃣ Thêm listener fd vào epoll
	// EPOLLIN: thông báo khi có client kết nối đến
	event := &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(sockFd),
	}
	if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, sockFd, event); err != nil {
		return err
	}

	log.Printf("listening on %s (fd=%d)\n", addr, sockFd)

	events := make([]syscall.EpollEvent, MaxEvents)

	// 6️⃣ Vòng event loop vô hạn
	for {
		// 6.1️⃣ Chờ events từ epoll (blocking)
		// - epoll_wait sẽ trả khi listener hoặc client có dữ liệu
		n, err := syscall.EpollWait(epfd, events, -1)
		if err != nil {
			if err == syscall.EINTR { // signal interrupt → tiếp tục
				continue
			}
			return err
		}

		// 6.2️⃣ Duyệt tất cả events vừa nhận
		for i := 0; i < n; i++ {
			ev := events[i]
			fd := int(ev.Fd)

			// 7️⃣ Nếu event đến từ listener → accept tất cả client mới
			if fd == sockFd {
				for {
					// 7.1️⃣ Accept connection mới (non-blocking)
					newFd, Sockaddr, err := syscall.Accept(sockFd)
					if err != nil {
						// Nếu không còn client chờ → EAGAIN/EWOULDBLOCK → dừng vòng accept
						if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
							break
						}
						log.Printf("accept err: %v\n", err)
						break
					}
					_ = Sockaddr // chưa dùng, có thể log client addr

					// 7.2️⃣ Đặt client socket non-blocking
					// => read/write sẽ không block event loop
					if err := syscall.SetNonblock(newFd, true); err != nil {
						log.Printf("setnonblock conn err: %v\n", err)
						syscall.Close(newFd)
						continue
					}

					// 7.3️⃣ Thêm client fd vào epoll
					// - EPOLLIN: có dữ liệu từ client
					// - EPOLLRDHUP: client đóng connection
					clientEvent := &syscall.EpollEvent{
						Events: syscall.EPOLLIN | syscall.EPOLLRDHUP,
						Fd:     int32(newFd),
					}
					if err := syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, newFd, clientEvent); err != nil {
						log.Printf("epoll ctl add conn err: %v\n", err)
						syscall.Close(newFd)
						continue
					}

					// 7.4️⃣ Khởi tạo buffer / state cho connection mới
					// - ConnBuf dùng lưu dữ liệu đọc/ghi cho client này
					epollServer.connsLock.Lock()
					epollServer.conns[newFd] = NewConnBuf()
					epollServer.connsLock.Unlock()
					log.Printf("accepted fd=%d\n", newFd)
				}

			} else {
				// 8️⃣ Nếu event đến từ client fd → xử lý read hoặc detect close
				if ev.Events&(syscall.EPOLLIN|syscall.EPOLLHUP|syscall.EPOLLRDHUP) != 0 {
					// Mỗi client có thể xử lý trong goroutine
					// go epollServer.handleRead(fd)
					epollServer.handleRead(fd) // debug
				}
			}
		}
	}
}

// handleRead: đọc dữ liệu từ fd và parse/execute nhiều command nếu có (pipelining)
func (epollServer *EpollServer) handleRead(fd int) {
	// 1. đọc bytes từ fd
	buf := make([]byte, 4096)
	n, err := syscall.Read(fd, buf)
	if err != nil {
		if err == syscall.EAGAIN {
			// 1.1: nothing to read
			return
		}
		// 1.2: error khác -> đóng connection
		epollServer.closeConn(fd)
		return
	}
	if n == 0 {
		// 2. client đóng connection
		epollServer.closeConn(fd)
		return
	}
	// 3. append data vào ConnBuf tương ứng
	epollServer.connsLock.Lock()
	cb := epollServer.conns[fd]
	epollServer.connsLock.Unlock()
	cb.appendData(buf[:n])

	// 4. thử parse nhiều command liên tiếp (pipelining)
	for {
		args, used, perr := parseRESPArray(cb.buf)
		if perr != nil {
			// 4.1: nếu incomplete -> chờ thêm dữ liệu
			if perr.Error() == "incomplete" {
				break
			}
			// 4.2: protocol error -> trả lỗi cho client và xóa buffer
			resp := respError("ERR " + perr.Error())
			syscall.Write(fd, resp)
			cb.buf = nil
			break
		}
		// 5. consume used bytes
		if used > 0 {
			cb.buf = cb.buf[used:]
		}
		// 6. thực thi command
		resp := epollServer.executeCommand(args)
		// 7. gửi response (ghi thẳng vào fd)
		_, werr := syscall.Write(fd, resp)
		if werr != nil {
			epollServer.closeConn(fd)
			return
		}
	}
}

// closeConn: xóa conn khỏi map và đóng fd
func (epollServer *EpollServer) closeConn(fd int) {
	// 1. lock map
	epollServer.connsLock.Lock()
	defer epollServer.connsLock.Unlock()
	// 2. xóa key nếu có
	if _, ok := epollServer.conns[fd]; ok {
		delete(epollServer.conns, fd)
	}
	// 3. đóng fd system
	syscall.Close(fd)
	log.Printf("closed fd=%d\n", fd)
}

// ---------- Command execution ----------
func (epollServer *EpollServer) executeCommand(args []string) []byte {
	// 1. kiểm tra args
	if len(args) == 0 {
		return respError("empty command")
	}
	// 2. switch command (case-insensitive)
	cmd := strings.ToUpper(args[0])
	switch cmd {
	case "PING":
		// 1. nếu có 1 arg nữa -> trả arg đó
		if len(args) == 2 {
			return respSimpleString(args[1])
		}
		// 2. mặc định trả PONG
		return respSimpleString("PONG")
	case "ECHO":
		// 1. cần đúng 1 arg
		if len(args) != 2 {
			return respError("wrong number of arguments for 'ECHO' command")
		}
		// 2. trả bulk string chứa message
		return respBulkString(args[1])
	case "SET":
		// SET key value [EX seconds]
		// 1. validate số arg tối thiểu
		if len(args) < 3 {
			return respError("wrong number of arguments for 'SET'")
		}
		// 2. lấy key/value
		key := args[1]
		val := args[2]
		// 3. parse optional flags (chỉ EX xử lý)
		exSec := 0
		if len(args) > 3 {
			for i := 3; i < len(args); i++ {
				part := strings.ToUpper(args[i])
				if part == "EX" && i+1 < len(args) {
					nsec, err := strconv.Atoi(args[i+1])
					if err != nil {
						return respError("value is not an integer or out of range")
					}
					exSec = nsec
					i++ // skip next arg
				}
			}
		}
		// 4. ghi vào store
		epollServer.store.setString(key, val, exSec)
		// 5. trả OK
		return respSimpleString("OK")
	case "GET":
		// 1. validate số arg
		if len(args) != 2 {
			return respError("wrong number of arguments for 'GET'")
		}
		// 2. lấy key và truy vấn store
		key := args[1]
		v, ok := epollServer.store.getString(key)
		if !ok {
			// 3. nếu không tồn tại -> null bulk
			return respNullBulk()
		}
		// 4. trả bulk string
		return respBulkString(v)
	case "TTL":
		// 1. validate
		if len(args) != 2 {
			return respError("wrong number of arguments for 'TTL'")
		}
		// 2. lấy ttl từ store
		key := args[1]
		ttl := epollServer.store.ttl(key)
		return respInteger(ttl)
	case "ZADD":
		// ZADD key score member
		// 1. validate
		if len(args) < 4 {
			return respError("wrong number of arguments for 'ZADD'")
		}
		// 2. parse score
		key := args[1]
		scoreStr := args[2]
		member := args[3]
		score, err := strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			return respError("ERR score is not a float")
		}
		// 3. đảm bảo zset tồn tại và add
		z := epollServer.store.ensureZSet(key)
		z.Add(score, member)
		// 4. trả số phần tử thêm thành công (đơn giản trả 1)
		return respInteger(1)
	case "ZRANGE":
		// ZRANGE key start stop [WITHSCORES]
		// 1. validate
		if len(args) < 4 {
			return respError("wrong number of arguments for 'ZRANGE'")
		}
		// 2. parse indices
		key := args[1]
		start, err1 := strconv.Atoi(args[2])
		stop, err2 := strconv.Atoi(args[3])
		if err1 != nil || err2 != nil {
			return respError("ERR start or stop is not an integer")
		}
		// 3. check optional WITHSCORES
		withScores := false
		if len(args) > 4 && strings.ToUpper(args[4]) == "WITHSCORES" {
			withScores = true
		}
		// 4. lấy zset
		z, ok := epollServer.store.getZSet(key)
		if !ok {
			return respArrayStrings(nil)
		}
		// 5. lấy range
		items := z.Range(start, stop)
		// 6. build response tuỳ WITHSCORES
		if withScores {
			out := make([]string, 0, len(items)*2)
			for _, it := range items {
				out = append(out, it.member)
				s := strconv.FormatFloat(it.score, 'f', -1, 64)
				out = append(out, s)
			}
			return respArrayStrings(out)
		}
		out := make([]string, 0, len(items))
		for _, it := range items {
			out = append(out, it.member)
		}
		return respArrayStrings(out)
	default:
		return respError("unknown command '" + args[0] + "'")
	}
}

// ---------- main ----------
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run miniredis_commented.go [addr:port], e.g. 0.0.0.0:6379")
		return
	}
	addr := os.Args[1]
	s := NewEpollServer()
	// background janitor để clear expired keys mỗi giây
	go func() {
		for {
			// 1. sleep 1s
			time.Sleep(1 * time.Second)
			// 2. quét map và xóa key expired
			now := time.Now()
			s.store.mu.Lock()
			for k, v := range s.store.m {
				if !v.expireAt.IsZero() && now.After(v.expireAt) {
					delete(s.store.m, k)
				}
			}
			s.store.mu.Unlock()
		}
	}()
	if err := s.Start(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
