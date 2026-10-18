package server

import (
	"html/template"
	"net/http"

	"github.com/lavantien/go-sudoku-solver/internal/logging"
	"github.com/lavantien/go-sudoku-solver/web"
)

const (
	templatesDir  = "templates/"
	layoutName    = "layout"
	staticDir     = "static"
	fragmentPfx   = "fragment_"
	solveErrorDef = "solve_error"
)

type templates struct {
	pages     map[string]*template.Template
	fragments map[string]*template.Template
}

func parseTemplates() *templates {
	pageSet := func(files ...string) *template.Template {
		base := template.Must(template.ParseFS(web.Templates, templatesDir+"layout.html"))
		clone := template.Must(base.Clone())
		paths := make([]string, 0, len(files))
		for _, f := range files {
			paths = append(paths, templatesDir+f)
		}
		return template.Must(clone.ParseFS(web.Templates, paths...))
	}
	return &templates{
		pages: map[string]*template.Template{
			"home":     pageSet("home.html"),
			"generate": pageSet("generate.html"),
			"history":  pageSet("history.html", "history_fragment.html"),
			"run":      pageSet("run.html", "solve_results.html"),
		},
		fragments: map[string]*template.Template{
			"solve":  template.Must(template.ParseFS(web.Templates, templatesDir+"solve_results.html")),
			"puzzle": template.Must(template.ParseFS(web.Templates, templatesDir+"solve_results.html", templatesDir+"puzzle_fragment.html")),
		},
	}
}

func (s *Server) renderPage(w http.ResponseWriter, name string, data any) {
	set, ok := s.templates.pages[name]
	if !ok {
		logging.Fail(s.log, "server.render_page", errTemplateNotFound(name))
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := set.ExecuteTemplate(w, layoutName, data); err != nil {
		logging.Fail(s.log, "server.render_page", err, "template", name)
	}
}

func (s *Server) renderFragment(w http.ResponseWriter, status int, name string, data any) {
	set, ok := s.templates.fragments[name]
	if !ok {
		logging.Fail(s.log, "server.render_fragment", errTemplateNotFound(name))
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := set.ExecuteTemplate(w, fragmentPfx+name, data); err != nil {
		logging.Fail(s.log, "server.render_fragment", err, "template", name)
	}
}

func (s *Server) renderSolveError(w http.ResponseWriter, status int, msg string) {
	set := s.templates.fragments["solve"]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := set.ExecuteTemplate(w, solveErrorDef, errorView{Message: msg}); err != nil {
		logging.Fail(s.log, "server.render_solve_error", err)
	}
}
