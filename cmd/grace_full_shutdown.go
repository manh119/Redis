package main

//
//import (
//	"io"
//	"log"
//	"net"
//	"os"
//	"os/signal"
//	"sync"
//	"sync/atomic"
//	"syscall"
//	"time"
//
//	"github.com/manh119/Redis/internal/core/config"
//	"github.com/manh119/Redis/internal/core/io_multiplexing"
//)
//
//// ============================================================
//// Graceful Shutdown — Redis-style (nghiên cứu từ src/server.c)
////
//// Redis thực tế làm gì:
////
//// 1. Signal handler CHỈ set một atomic flag: server.shutdown_asap = 1
////    Không làm gì khác — signal handler phải async-signal-safe.
////
//// 2. Event loop tick (serverCron, chạy ~10 lần/giây) kiểm tra flag đó.
////    Nếu thấy shutdown_asap=1 → gọi prepareForShutdown().
////
//// 3. prepareForShutdown() thực hiện theo thứ tự:
////    a. pauseClients()          — dừng nhận command mới từ client
////    b. waitForReplicas()       — đợi replica sync (nếu có, tối đa 10s)
////    c. killChildProcesses()    — kill background RDB/AOF child
////    d. flushAppendOnlyFile()   — fsync AOF buffer ra disk
////    e. rdbSave()               — blocking RDB snapshot
////    f. removePidFile()         — xóa pid file
////    g. removeUnixSocket()      — xóa unix socket
////    h. exit(0)
////
//// 4. Nếu RDB save thất bại → KHÔNG exit, server tiếp tục chạy.
////    Phải nhận SIGTERM lần 2 mới thoát forced.
////
//// 5. "Scheduled shutdown": signal đến TRONG lúc đang execute command
////    → đợi command xong (tối đa 0.1s) → mới bắt đầu shutdown.
////    Đây chính là "graceful" — không cắt ngang command.
////
//// Map sang Go clone (single-threaded, không có RDB/AOF):
////   shutdown_asap     → atomic int32 shutdownFlag
////   serverCron tick   → kiểm tra ở đầu/cuối event loop iteration
////   prepareForShutdown → hàm prepareForShutdown() bên dưới
////   pauseClients      → close(shutdownCh) broadcast, dừng accept
////   waitForReplicas   → (bỏ qua, không có replica)
////   flushAOF/rdbSave  → persistence.Save() nếu có
////   exit(0)           → os.Exit(0) sau khi cleanup defer đã chạy
//// ============================================================
//
//// ── Trạng thái shutdown (giống server.shutdown_asap trong Redis) ──────────
//
//// shutdownAsap được set = 1 bởi signal handler, đọc bởi event loop.
//// Chỉ dùng atomic — KHÔNG dùng mutex, giống Redis dùng atomicSet/atomicGet.
//var shutdownAsap atomic.Int32
//
//// shutdownCh được close() khi shutdown bắt đầu → broadcast tới mọi goroutine.
//// Đây là Go-idiom tương đương "clientPauseAllClients" của Redis:
//// sau khi close, tất cả select { case <-shutdownCh } đều unblock ngay lập tức.
//var shutdownCh = make(chan struct{})
//var shutdownOnce sync.Once
//
//const (
//	// Giống shutdown-timeout mặc định của Redis = 10s
//	ShutdownTimeout = 10 * time.Second
//)
//
//// ── Entry point ───────────────────────────────────────────────────────────
//
//func Main() {
//	// Signal handler CHỈ set flag, không làm gì thêm.
//	// Đây đúng với Redis: signal handler phải async-signal-safe,
//	// không được gọi malloc, log, hay bất cứ thứ gì phức tạp.
//	sigs := make(chan os.Signal, 1)
//	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
//
//	go func() {
//		sig := <-sigs
//		// Chỉ set flag — event loop sẽ xử lý ở iteration tiếp theo.
//		// Giống: atomicSet(server.shutdown_asap, 1) trong sigHandler().
//		log.Printf("[signal] received %v, scheduling shutdown", sig)
//		shutdownAsap.Store(1)
//	}()
//
//	var wg sync.WaitGroup
//	wg.Add(1)
//	go RunIoMultiplexingServer(&wg)
//	wg.Wait()
//}
//
//// ── Event loop chính ──────────────────────────────────────────────────────
//
//func RunIoMultiplexingServer(wg *sync.WaitGroup) {
//	defer wg.Done()
//
//	log.Println("Starting I/O Multiplexing TCP server on", config.Port)
//
//	listener, err := net.Listen(config.Protocol, config.Port)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer listener.Close()
//
//	tcpListener := listener.(*net.TCPListener)
//	listenerFile, err := tcpListener.File()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer listenerFile.Close()
//	serverFd := int(listenerFile.Fd())
//
//	ioMultiplexer, err := io_multiplexing.CreateIOMultiplexer()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer ioMultiplexer.Close()
//
//	if err = ioMultiplexer.Monitor(io_multiplexing.Event{Fd: serverFd, Op: io_multiplexing.OpRead}); err != nil {
//		log.Fatal(err)
//	}
//
//	// Active expiry chạy trong goroutine riêng, thoát khi shutdownCh bị close.
//	go runActiveExpiry()
//
//	events := make([]io_multiplexing.Event, config.MaxConnection)
//	var lastExpireTime = time.Now()
//
//	for {
//		// ── Checkpoint ① — Giống serverCron() check shutdown_asap ──────
//		// Redis kiểm tra ở đầu mỗi cron tick (~100ms).
//		// Chúng ta kiểm tra ở đầu mỗi event loop iteration.
//		if shutdownAsap.Load() == 1 {
//			// "Scheduled shutdown starts as soon as possible, specifically
//			// as long as the current command in execution terminates."
//			// → Command đang chạy đã xong (chúng ta single-threaded),
//			// nên bắt đầu shutdown ngay.
//			prepareForShutdown(listener, ioMultiplexer)
//			return
//		}
//
//		// Active expiry (tách khỏi event loop chính, không block)
//		if time.Since(lastExpireTime) >= constant.ActiveExpireFrequency {
//			lastExpireTime = time.Now()
//			// Chạy trong goroutine riêng để không block event loop.
//			// Redis cũng làm tương tự: activeExpireCycle() trong serverCron.
//		}
//
//		events, err = ioMultiplexer.Wait()
//		if err != nil {
//			// ioMultiplexer.Wait() có thể trả lỗi khi fd bị đóng lúc shutdown.
//			if shutdownAsap.Load() == 1 {
//				prepareForShutdown(listener, ioMultiplexer)
//				return
//			}
//			log.Println("epoll wait error:", err)
//			continue
//		}
//
//		// ── Checkpoint ② — Sau khi Wait() unblock ──────────────────────
//		// Giống Redis check giữa các command trong pipeline.
//		if shutdownAsap.Load() == 1 {
//			prepareForShutdown(listener, ioMultiplexer)
//			return
//		}
//
//		for i := range events {
//			if events[i].Fd == serverFd {
//				handleNewConnection(serverFd, ioMultiplexer)
//			} else {
//				handleClientCommand(events[i].Fd)
//			}
//		}
//
//		// ── Checkpoint ③ — Sau khi xử lý hết batch events ─────────────
//		// Tương đương afterSleep() callback của Redis:
//		// Redis gọi các hook sau mỗi lần epoll_wait trả về.
//		if shutdownAsap.Load() == 1 {
//			prepareForShutdown(listener, ioMultiplexer)
//			return
//		}
//	}
//}
//
//// ── prepareForShutdown — ánh xạ 1:1 với Redis prepareForShutdown() ────────
//
//// prepareForShutdown thực hiện đúng sequence Redis dùng:
//// pause → (wait replicas) → flush AOF → save RDB → cleanup → exit
////
//// Khác với os.Exit(0) trong signal handler:
//// hàm này chạy trong event loop goroutine → tất cả defer ở caller đã
//// được schedule → sau khi return, defer listener.Close() v.v. sẽ chạy.
//func prepareForShutdown(listener net.Listener, ioMultiplexer io_multiplexing.IOMultiplexer) {
//	shutdownOnce.Do(func() {
//		log.Println("Initiating graceful shutdown (Redis-style)...")
//
//		// Bước 1: Pause clients — dừng nhận request mới.
//		// Redis: clientPauseAllClients(PAUSE_WRITE, ...)
//		// Go: close(shutdownCh) broadcast + đóng listener sớm.
//		pauseClients(listener)
//
//		// Bước 2: Đợi command đang thực thi xong.
//		// Redis: "as long as the current command in execution terminates"
//		// Chúng ta single-threaded nên khi đến đây, command đã xong rồi.
//		// Nếu sau này multi-threaded, thêm: waitForActiveCommands()
//
//		// Bước 3: Flush persistence (nếu có AOF/RDB).
//		// Redis: flushAppendOnlyFile(1) → rdbSave()
//		// Nếu save thất bại, Redis KHÔNG exit → server tiếp tục chạy.
//		if err := persistence.Save(); err != nil {
//			log.Printf("WARNING: persistence save failed: %v", err)
//			log.Println("Shutdown aborted to prevent data loss. Send SIGTERM again to force.")
//			// Reset flag để server tiếp tục chạy — đúng với Redis behavior.
//			shutdownAsap.Store(0)
//			return
//		}
//
//		// Bước 4: Dọn dẹp tài nguyên.
//		// Redis: removePidFile(), removeUnixSocket()
//		cleanup()
//
//		log.Println("Graceful shutdown complete.")
//		// Bước 5: Exit với code 0.
//		// defer trong RunIoMultiplexingServer đã chạy trước khi hàm này
//		// được gọi (vì chúng ta return trước khi defer chạy)...
//		// KHÔNG — defer chỉ chạy khi goroutine return.
//		// Ở đây chúng ta os.Exit sau khi đã cleanup thủ công.
//		// Listener và ioMultiplexer đã bị close trong pauseClients() rồi.
//		os.Exit(0)
//	})
//}
//
//// pauseClients dừng nhận connection/command mới.
//// Tương đương clientPauseAllClients() + close listening socket của Redis.
//func pauseClients(listener net.Listener) {
//	log.Println("Pausing clients (no new connections accepted)...")
//	// Close listener ngay → kernel từ chối connection mới với ECONNREFUSED.
//	// Clients đang kết nối sẽ nhận EOF khi server đóng fd của họ.
//	if err := listener.Close(); err != nil {
//		log.Println("listener close error:", err)
//	}
//	// Broadcast shutdown tới tất cả goroutine (active expiry, v.v.)
//	close(shutdownCh)
//}
//
//// cleanup xóa pid file, unix socket, v.v.
//func cleanup() {
//	if config.PidFile != "" {
//		if err := os.Remove(config.PidFile); err != nil && !os.IsNotExist(err) {
//			log.Printf("WARNING: cannot remove pid file %s: %v", config.PidFile, err)
//		}
//	}
//	if config.UnixSocket != "" {
//		if err := os.Remove(config.UnixSocket); err != nil && !os.IsNotExist(err) {
//			log.Printf("WARNING: cannot remove unix socket %s: %v", config.UnixSocket, err)
//		}
//	}
//}
//
//// ── Connection & command handlers ─────────────────────────────────────────
//
//func handleNewConnection(serverFd int, ioMultiplexer io_multiplexing.IOMultiplexer) {
//	// Nếu shutdown đang diễn ra, từ chối connection mới.
//	// Redis: sau khi listener bị close, kernel tự từ chối — chúng ta
//	// không cần check ở đây. Nhưng giữ check để an toàn.
//	if shutdownAsap.Load() == 1 {
//		// Vẫn accept để lấy fd, rồi gửi error và đóng ngay.
//		connFd, _, err := syscall.Accept(serverFd)
//		if err != nil {
//			return
//		}
//		// Gửi Redis inline error — client sẽ hiểu server đang shutdown.
//		writeRedisError(connFd, "LOADING Redis is starting")
//		syscall.Close(connFd)
//		return
//	}
//
//	connFd, _, err := syscall.Accept(serverFd)
//	if err != nil {
//		log.Println("accept error:", err)
//		return
//	}
//	log.Printf("New connection fd=%d", connFd)
//
//	if err = ioMultiplexer.Monitor(io_multiplexing.Event{Fd: connFd, Op: io_multiplexing.OpRead}); err != nil {
//		log.Println("monitor error:", err)
//		syscall.Close(connFd)
//	}
//}
//
//func handleClientCommand(fd int) {
//	cmd, err := readCommand(fd)
//	if err != nil {
//		if err == io.EOF || err == syscall.ECONNRESET {
//			log.Printf("Client fd=%d disconnected", fd)
//		} else {
//			log.Printf("Read error fd=%d: %v", fd, err)
//		}
//		syscall.Close(fd)
//		return
//	}
//
//	if err = core.ExecuteAndResponse(cmd, fd); err != nil {
//		log.Printf("Write error fd=%d: %v", fd, err)
//	}
//}
//
//// writeRedisError gửi Redis Simple Error response.
//// Format RESP: "-ERR message\r\n"
//func writeRedisError(fd int, msg string) {
//	resp := "-" + msg + "\r\n"
//	_, _ = syscall.Write(fd, []byte(resp))
//}
//
//// ── Active expiry goroutine ───────────────────────────────────────────────
//
//// runActiveExpiry chạy active key expiration định kỳ.
//// Redis: activeExpireCycle() được gọi trong serverCron().
//// Goroutine này thoát clean khi shutdownCh bị close.
//func runActiveExpiry() {
//	ticker := time.NewTicker(constant.ActiveExpireFrequency)
//	defer ticker.Stop()
//	for {
//		select {
//		case <-shutdownCh:
//			log.Println("Active expiry goroutine: shutting down.")
//			return
//		case <-ticker.C:
//			core.ActiveDeleteExpiredKeys()
//		}
//	}
//}
