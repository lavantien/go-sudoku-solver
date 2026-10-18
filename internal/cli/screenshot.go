package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/lavantien/go-sudoku-solver/internal/app"
	"github.com/lavantien/go-sudoku-solver/internal/config"
	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/internal/puzzle"
	"github.com/lavantien/go-sudoku-solver/internal/server"
	"github.com/lavantien/go-sudoku-solver/internal/store"
)

func init() {
	Register(Command{Name: config.CmdScreenshot, Usage: config.UsageScreenshot, Run: runScreenshot})
}

func runScreenshot(ctx context.Context, cfg config.Config, args []string) error {
	logger := logging.New(cfg.LogLevel)
	logging.Enter(logger, config.CmdScreenshot)
	fs := flag.NewFlagSet(config.CmdScreenshot, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	out := fs.String(config.FlagOut, config.DefaultPicturesDir, config.FlagOutUsage)
	if err := fs.Parse(args); err != nil {
		logging.Fail(logger, config.CmdScreenshot, err)
		return err
	}
	if fs.NArg() > 0 {
		err := fmt.Errorf("unexpected argument %q", fs.Arg(0))
		logging.Fail(logger, config.CmdScreenshot, err)
		return err
	}
	if err := screenshotRun(ctx, cfg, *out, logger); err != nil {
		logging.Fail(logger, config.CmdScreenshot, err, "out", *out)
		return err
	}
	logging.Exit(logger, config.CmdScreenshot, "out", *out)
	return nil
}

func screenshotRun(ctx context.Context, cfg config.Config, outDir string, logger *slog.Logger) error {
	logging.Enter(logger, "screenshot.run", "out", outDir)
	browser, err := detectBrowser()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	site, err := bootSite(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer site.shutdown()
	apiBody, err := postSolve(site.base)
	if err != nil {
		return err
	}
	logging.Exit(logger, "screenshot.api", "bytes", len(apiBody))
	cliOut, err := captureCLI(ctx, cfg)
	if err != nil {
		return err
	}
	logging.Exit(logger, "screenshot.cli", "bytes", len(cliOut))
	if err := shoot(ctx, browser, site.base, apiBody, cliOut, outDir, logger); err != nil {
		return err
	}
	logging.Exit(logger, "screenshot.run", "browser", browser)
	return nil
}

type site struct {
	base     string
	shutdown func()
}

func bootSite(ctx context.Context, cfg config.Config, logger *slog.Logger) (*site, error) {
	logging.Enter(logger, "screenshot.site")
	tmpDir, err := os.MkdirTemp("", "sudoku-screenshot-*")
	if err != nil {
		return nil, err
	}
	st, err := store.Open(filepath.Join(tmpDir, "site.db"))
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}
	srv := server.New(ctx, cfg, st, logger)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = st.Close()
		_ = os.RemoveAll(tmpDir)
		return nil, err
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	stop := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logging.Fail(logger, "screenshot.site", err, "step", "shutdown")
		}
		if err := <-serveErr; err != nil {
			logging.Fail(logger, "screenshot.site", err, "step", "serve")
		}
		_ = st.Close()
		_ = os.RemoveAll(tmpDir)
		logging.Exit(logger, "screenshot.site")
	}
	return &site{base: "http://" + ln.Addr().String(), shutdown: stop}, nil
}

func postSolve(base string) (string, error) {
	reqBody := fmt.Sprintf(`{"%s": %q}`, config.FormGrid81, puzzle.Sample81)
	resp, err := http.Post(base+config.APISolvePath, config.JSONContentType, strings.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("api solve status %d: %s", resp.StatusCode, raw)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return "", err
	}
	return pretty.String(), nil
}

func captureCLI(ctx context.Context, cfg config.Config) (string, error) {
	rows := make([]string, 0, config.Size)
	for r := range config.Size {
		rows = append(rows, puzzle.Sample81[r*config.Size:(r+1)*config.Size])
	}
	var b strings.Builder
	if code := app.Run(ctx, cfg, strings.NewReader(strings.Join(rows, "\n")), &b, io.Discard); code != 0 {
		return "", fmt.Errorf("cli solve exited %d", code)
	}
	return b.String(), nil
}

func detectBrowser() (string, error) {
	if p := os.Getenv(config.BrowserEnv); p != "" {
		if info, err := os.Stat(p); err != nil || info.IsDir() {
			return "", fmt.Errorf("%s path %q is not an executable file", config.BrowserEnv, p)
		}
		return p, nil
	}
	for _, cand := range browserCandidates() {
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand, nil
		}
	}
	return "", fmt.Errorf("no chrome or edge executable found, set %s to a browser path", config.BrowserEnv)
}

func browserCandidates() []string {
	chrome := browserTier(
		[]string{"chrome", "google-chrome", "google-chrome-stable", "chromium", "chromium-browser"},
		chromePaths(),
	)
	edge := browserTier(
		[]string{"msedge", "microsoft-edge"},
		edgePaths(),
	)
	return append(chrome, edge...)
}

func browserTier(names, paths []string) []string {
	tier := make([]string, 0, len(names)+len(paths))
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			tier = append(tier, p)
		}
	}
	return append(tier, paths...)
}

func chromePaths() []string {
	switch runtime.GOOS {
	case "windows":
		return joinEnvDirs([]string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"},
			filepath.Join("Google", "Chrome", "Application", "chrome.exe"))
	case "darwin":
		return []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			filepath.Join(os.Getenv("HOME"), "Applications", "Google Chrome.app", "Contents", "MacOS", "Google Chrome"),
		}
	default:
		return []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		}
	}
}

func edgePaths() []string {
	switch runtime.GOOS {
	case "windows":
		return joinEnvDirs([]string{"ProgramFiles", "ProgramFiles(x86)"},
			filepath.Join("Microsoft", "Edge", "Application", "msedge.exe"))
	case "darwin":
		return []string{"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"}
	default:
		return []string{"/usr/bin/microsoft-edge", "/usr/bin/microsoft-edge-stable"}
	}
}

func joinEnvDirs(vars []string, rel string) []string {
	out := make([]string, 0, len(vars))
	for _, v := range vars {
		if d := os.Getenv(v); d != "" {
			out = append(out, filepath.Join(d, rel))
		}
	}
	return out
}

func shoot(ctx context.Context, browser, base, apiBody, cliOut, outDir string, logger *slog.Logger) error {
	logging.Enter(logger, "screenshot.shoot", "browser", browser, "base", base)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browser),
		chromedp.Flag("disable-gpu", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	bctx, cancelBrowse := chromedp.NewContext(allocCtx)
	defer cancelBrowse()
	shotCtx, cancelBudget := context.WithTimeout(bctx, config.ShotBudget)
	defer cancelBudget()
	s := &shotter{ctx: shotCtx, outDir: outDir, logger: logger}
	if err := chromedp.Do(shotCtx, chromedp.EmulateViewport(config.ShotWidth, config.ShotHeight)); err != nil {
		return err
	}
	if err := s.homeAndSolve(base); err != nil {
		return err
	}
	if err := s.generator(base); err != nil {
		return err
	}
	if err := s.history(base); err != nil {
		return err
	}
	if err := s.runDetail(base); err != nil {
		return err
	}
	if err := s.apiJSON(apiBody); err != nil {
		return err
	}
	if err := s.cliSession(cliOut); err != nil {
		return err
	}
	if err := s.curlSession(apiBody); err != nil {
		return err
	}
	logging.Exit(logger, "screenshot.shoot")
	return nil
}

type shotter struct {
	ctx    context.Context
	outDir string
	logger *slog.Logger
	runRef string
}

func (s *shotter) full(name string) error {
	logging.Enter(s.logger, "screenshot.shot", "file", name)
	png, err := chromedp.Run(s.ctx, chromedp.FullScreenshot(100))
	if err != nil {
		return err
	}
	if len(png) == 0 {
		return fmt.Errorf("empty screenshot for %s", name)
	}
	path := filepath.Join(s.outDir, name)
	if err := os.WriteFile(path, png, 0o644); err != nil {
		return err
	}
	fmt.Printf("captured %s (%d bytes)\n", path, len(png))
	logging.Exit(s.logger, "screenshot.shot", "file", name, "bytes", len(png))
	return nil
}

func (s *shotter) homeAndSolve(base string) error {
	if err := chromedp.Do(s.ctx,
		chromedp.Navigate(base+config.HomePath),
		chromedp.WaitVisible("#grid-input"),
	); err != nil {
		return err
	}
	if err := s.full(config.PictureHome); err != nil {
		return err
	}
	if err := chromedp.Do(s.ctx,
		chromedp.Click(".sample-btn"),
		chromedp.Click("button.primary"),
		chromedp.WaitVisible("#results .run-link a"),
	); err != nil {
		return err
	}
	ref, err := chromedp.Run(s.ctx, chromedp.AttributeValue("#results .run-link a", "href"))
	if err != nil {
		return err
	}
	if !ref.Exists || ref.Value == "" {
		return fmt.Errorf("solve fragment lacks the run link")
	}
	s.runRef = ref.Value
	return s.full(config.PictureSolve)
}

func (s *shotter) generator(base string) error {
	if err := chromedp.Do(s.ctx,
		chromedp.Navigate(base+config.GeneratePagePath),
		chromedp.Click("button.primary"),
		chromedp.WaitVisible("#puzzle table.grid"),
	); err != nil {
		return err
	}
	return s.full(config.PictureGenerator)
}

func (s *shotter) history(base string) error {
	if err := chromedp.Do(s.ctx,
		chromedp.Navigate(base+config.HistoryPagePath),
		chromedp.WaitVisible("table.runs tbody tr"),
	); err != nil {
		return err
	}
	return s.full(config.PictureHistory)
}

func (s *shotter) runDetail(base string) error {
	if err := chromedp.Do(s.ctx,
		chromedp.Navigate(base+s.runRef),
		chromedp.WaitVisible(".cards .card"),
	); err != nil {
		return err
	}
	return s.full(config.PictureRunDetail)
}

func (s *shotter) apiJSON(apiBody string) error {
	reqLine := "POST " + config.APISolvePath
	doc := pageDoc("api json", lightCSS, "<main><h1>api json</h1><p class=\"req\">"+html.EscapeString(reqLine)+"</p><pre>"+html.EscapeString(apiBody)+"</pre></main>")
	if err := chromedp.Do(s.ctx, chromedp.Navigate(dataURL(doc))); err != nil {
		return err
	}
	return s.full(config.PictureAPIJSON)
}

func (s *shotter) cliSession(cliOut string) error {
	doc := terminalDoc("cli session", config.CliPrompt+"\n"+cliOut)
	if err := chromedp.Do(s.ctx, chromedp.Navigate(dataURL(doc))); err != nil {
		return err
	}
	return s.full(config.PictureCLISession)
}

func (s *shotter) curlSession(apiBody string) error {
	cmd := fmt.Sprintf(config.CurlPromptFmt, config.Defaults().Server.Addr, config.APISolvePath, config.JSONContentType, config.FormGrid81, puzzle.Sample81)
	doc := terminalDoc("curl session", cmd+"\n\n"+apiBody)
	if err := chromedp.Do(s.ctx, chromedp.Navigate(dataURL(doc))); err != nil {
		return err
	}
	return s.full(config.PictureCurlCard)
}

const darkCSS = `*{margin:0;padding:0;box-sizing:border-box}
body{background:#0b0e13;padding:32px;min-height:100vh}
.term{max-width:1080px;margin:0 auto;background:#11151c;border:1px solid #2a3240;border-radius:10px;overflow:hidden}
.bar{display:flex;align-items:center;gap:8px;padding:10px 14px;background:#1a2029;border-bottom:1px solid #2a3240}
.dot{width:12px;height:12px;border-radius:50%}
.r{background:#ff5f56}.y{background:#ffbd2e}.g{background:#27c93f}
.ttl{margin-left:12px;color:#8b98a9;font:14px system-ui,sans-serif}
pre{padding:20px 24px;color:#d7e1ea;font:14px/1.55 ui-monospace,Consolas,monospace;white-space:pre-wrap;word-break:break-word}
`

const lightCSS = `*{margin:0;padding:0;box-sizing:border-box}
body{background:#f4f2ec;padding:32px;min-height:100vh;font:16px system-ui,sans-serif;color:#22252a}
main{max-width:900px;margin:0 auto;background:#fffdf8;border:1px solid #ddd8cc;border-radius:10px;padding:28px 32px}
h1{font-size:24px;margin-bottom:8px}
.req{font:14px ui-monospace,Consolas,monospace;color:#6b6455;margin-bottom:16px}
pre{font:14px/1.6 ui-monospace,Consolas,monospace;background:#22252a;color:#d9e2ec;padding:20px;border-radius:8px;overflow-x:auto}
`

func terminalDoc(title, body string) string {
	bar := "<header class=\"bar\"><span class=\"dot r\"></span><span class=\"dot y\"></span><span class=\"dot g\"></span><span class=\"ttl\">" + html.EscapeString(title) + "</span></header>"
	return pageDoc(title, darkCSS, "<main class=\"term\">"+bar+"<pre>"+html.EscapeString(body)+"</pre></main>")
}

func pageDoc(title, css, bodyHTML string) string {
	return "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>" + html.EscapeString(title) +
		"</title><style>" + css + "</style></head><body>" + bodyHTML + "</body></html>"
}

func dataURL(doc string) string {
	return "data:text/html;charset=utf-8," + strings.ReplaceAll(url.QueryEscape(doc), "+", "%20")
}
