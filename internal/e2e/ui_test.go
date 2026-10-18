package e2e

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/server"
	"github.com/lavantien/go-sudoku-solver/internal/store"
)

const uiTimeout = 3 * time.Minute

type cardJSON struct {
	Label  string `json:"label"`
	Digits string `json:"digits"`
	Solved bool   `json:"solved"`
}

const cardsExpr = `JSON.stringify(Array.from(document.querySelectorAll('#results .cards article.card')).map(function (c) {
	return {
		label: c.querySelector('h2').textContent,
		digits: Array.from(c.querySelectorAll('table.grid td')).filter(function (td) {
			return !td.classList.contains('pipe');
		}).map(function (td) {
			return td.textContent;
		}).join(''),
		solved: c.classList.contains('is-solved')
	};
}))`

func TestUISolveJourney(t *testing.T) {
	if os.Getenv(envUI) != "1" {
		t.Skipf("ui gate off: set %s=1 or run make e2e", envUI)
	}
	browser := detectBrowser(t)
	t.Logf("e2e: ui browser %s", browser)
	addr := bootInProcessServe(t)
	typed := nineLineInput(puzzle.Sample81)

	opts := append(slices.Clone(chromedp.DefaultExecAllocatorOptions[:]), chromedp.ExecPath(browser))
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()
	ctx, cancelTimeout := context.WithTimeout(browserCtx, uiTimeout)
	defer cancelTimeout()

	if err := chromedp.Do(ctx,
		chromedp.Navigate("http://"+addr+config.HomePath),
		chromedp.WaitVisible(`#grid-input`),
		chromedp.Evaluate[chromedp.Void](`document.getElementById('grid-input').value = ''`),
		chromedp.SendKeys(`#grid-input`, typed),
	); err != nil {
		t.Fatalf("navigate and type: %v", err)
	}
	typedValue, err := chromedp.Run(ctx, chromedp.Value(`#grid-input`))
	if err != nil {
		t.Fatalf("read textarea: %v", err)
	}
	if typedValue != typed {
		t.Fatalf("textarea: got %q want the typed sample", typedValue)
	}
	if err := chromedp.Do(ctx, chromedp.Click(`button.primary`)); err != nil {
		t.Fatalf("click solve: %v", err)
	}
	if err := chromedp.Do(ctx, chromedp.WaitVisible(`#results .cards article.card:nth-of-type(4) table.grid`)); err != nil {
		t.Fatalf("wait for the 4 method cards: %v", err)
	}
	rawCards, err := chromedp.Run(ctx, chromedp.Evaluate[string](cardsExpr))
	if err != nil {
		t.Fatalf("collect cards: %v", err)
	}
	var cards []cardJSON
	if err := json.Unmarshal([]byte(rawCards), &cards); err != nil {
		t.Fatalf("decode cards %q: %v", rawCards, err)
	}
	if len(cards) != len(config.MethodIDs) {
		t.Fatalf("cards: got %d want %d", len(cards), len(config.MethodIDs))
	}
	for i, card := range cards {
		wantLabel := config.MethodLabel(config.MethodIDs[i])
		if card.Label != wantLabel {
			t.Errorf("card %d label: got %q want %q", i, card.Label, wantLabel)
		}
		if card.Label == config.MethodLabel(config.MethodIDs[2]) {
			if card.Solved {
				t.Errorf("anneal card shows solved")
			}
			if card.Digits == puzzle.SampleSolution81 {
				t.Errorf("anneal card shows the exact sample solution")
			}
			continue
		}
		if card.Digits != puzzle.SampleSolution81 {
			t.Errorf("card %q digits: got %s want the sample solution", card.Label, card.Digits)
		}
		if !card.Solved {
			t.Errorf("card %q shows unsolved", card.Label)
		}
	}
	shot, err := chromedp.Run(ctx, chromedp.FullScreenshot(90))
	if err != nil {
		t.Fatalf("screenshot: %v", err)
	}
	path := filepath.Join(t.TempDir(), "ui-journey.png")
	if err := os.WriteFile(path, shot, 0o644); err != nil {
		t.Fatalf("write screenshot: %v", err)
	}
	if len(shot) == 0 {
		t.Fatalf("screenshot at %s is empty", path)
	}
	t.Logf("e2e: ui screenshot written to %s (%d bytes)", path, len(shot))
}

func detectBrowser(t *testing.T) string {
	t.Helper()
	if p := os.Getenv(envBrowser); p != "" {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s=%q: %v", envBrowser, p, err)
		}
		return p
	}
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		}
		for _, c := range candidates {
			if c == "" {
				continue
			}
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	} else {
		candidates = []string{"google-chrome", "google-chrome-stable", "chromium", "chrome", "microsoft-edge"}
		for _, c := range candidates {
			if p, err := exec.LookPath(c); err == nil {
				return p
			}
		}
	}
	t.Fatalf("%s=1 but no browser was found, tried %v (override with %s)", envUI, candidates, envBrowser)
	return ""
}

func bootInProcessServe(t *testing.T) string {
	t.Helper()
	db := filepath.Join(t.TempDir(), "ui.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	srv := server.New(ctx, config.Defaults(), st, logging.New(""))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		_ = st.Close()
		t.Fatal(err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	t.Logf("e2e: in-process serve on %s", ln.Addr().String())
	t.Cleanup(func() {
		shutdownCtx, done := context.WithTimeout(context.Background(), config.Defaults().Server.ShutdownTimeout)
		_ = srv.Shutdown(shutdownCtx)
		done()
		cancel()
		if err := <-serveErr; err != nil {
			t.Errorf("serve returned: %v", err)
		}
		_ = st.Close()
	})
	return ln.Addr().String()
}
