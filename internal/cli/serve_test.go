package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lavantien/go-sudoku-solver/internal/config"
)

func bootServe(t *testing.T, args []string) (string, func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runServe(ctx, config.Defaults(), args) }()
	addrCh := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			if line, ok := strings.CutPrefix(sc.Text(), config.ServeAddrPrefix); ok {
				addrCh <- strings.TrimSpace(line)
				return
			}
		}
		if err := sc.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "serve: stderr scan: %v\n", err)
		}
		close(addrCh)
	}()
	var addr string
	select {
	case v, ok := <-addrCh:
		if !ok {
			os.Stderr = old
			t.Fatal("stderr closed without the addr line")
		}
		addr = v
	case err := <-done:
		os.Stderr = old
		t.Fatalf("serve exited before the addr line: %v", err)
	case <-time.After(60 * time.Second):
		os.Stderr = old
		t.Fatal("timeout waiting for the addr line")
	}
	stop := func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("serve returned: %v", err)
			}
		case <-time.After(60 * time.Second):
			t.Error("timeout waiting for serve to stop")
		}
		os.Stderr = old
		_ = w.Close()
		_ = r.Close()
	}
	return addr, stop
}

func TestServeBootsOnChosenPortAndServesAPI(t *testing.T) {
	db := filepath.Join(t.TempDir(), "serve.db")
	addr, stop := bootServe(t, []string{"--" + config.FlagAddr, "127.0.0.1:0", "--" + config.FlagDB, db})
	defer stop()
	if addr == config.Defaults().Server.Addr {
		t.Fatalf("addr line reports the default addr %q, want the chosen port", addr)
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		t.Fatalf("addr line is not host:port: %q", addr)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	defer client.CloseIdleConnections()
	resp, err := client.Get("http://" + addr + config.APIHealthPath)
	if err != nil {
		t.Fatalf("health on chosen addr: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status: got %d want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("health body %q: %v", raw, err)
	}
	if body["status"] != config.HealthStatus {
		t.Fatalf("health body: got %q want status %q", raw, config.HealthStatus)
	}
	if _, err := os.Stat(db); err != nil {
		t.Errorf("serve did not create the db at %s: %v", db, err)
	}
}

func TestServeRejectsBadFlagsAndBadAddr(t *testing.T) {
	if err := runServe(context.Background(), config.Defaults(), []string{"--bogus"}); err == nil {
		t.Error("unknown flag: got nil want error")
	}
	if err := runServe(context.Background(), config.Defaults(), []string{"--" + config.FlagAddr, "512.512.512.512:1", "--" + config.FlagDB, filepath.Join(t.TempDir(), "x.db")}); err == nil {
		t.Error("unlistenable addr: got nil want error")
	}
}

func TestServeRejectsEmptyAddrAndUnusableDB(t *testing.T) {
	if err := runServe(context.Background(), config.Defaults(), []string{"--" + config.FlagAddr, "", "--" + config.FlagDB, filepath.Join(t.TempDir(), "x.db")}); err == nil {
		t.Error("empty addr: got nil want error")
	}
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runServe(context.Background(), config.Defaults(), []string{"--" + config.FlagAddr, "127.0.0.1:0", "--" + config.FlagDB, filepath.Join(blocker, "x.db")}); err == nil {
		t.Error("db under a plain file: got nil want error")
	}
}

func TestServeAddrLineFmtMatchesPrefix(t *testing.T) {
	line := fmt.Sprintf(config.ServeAddrLineFmt, config.BinName, "127.0.0.1:1")
	if !strings.HasPrefix(line, config.ServeAddrPrefix) {
		t.Errorf("addr line %q lacks prefix %q", line, config.ServeAddrPrefix)
	}
}
