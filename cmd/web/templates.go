package main

import (
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"time"

	"zeeshan.kineta.site/internal/model"
	"zeeshan.kineta.site/ui"
)

type Template struct {
	User            *model.User
	Form            any
	Rating          any
	Post            *model.Post
	Posts           []*model.Post
	Users           []*model.User
	Flash           string
	IsAuthenticated bool
	IsActivated     bool
	IsOwner         bool
	HasRated        bool
	ShowNav         bool
	ShowSearcbar    bool
	DCExcedded      bool
	Status          int
}

type EmailTemplates struct {
	Name            string
	Id              string
	Token           string
	attempts        int
	IsAuthenticated bool
	IsActivated     bool
}

func HumanDate(t time.Time) string {
	return t.Format("02 Jan 2006 at 15:04")
}

var funcs = template.FuncMap{
	"humandate": HumanDate,
}

func NewTemplateCache() (map[string]*template.Template, error) {

	cache := map[string]*template.Template{}

	pages, err := fs.Glob(ui.Files, "html/pages/*.html")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		pattern := []string{
			"html/base.html",
			"html/partials/nav.html",
			"html/partials/searchbar.html",
			page,
		}
		ts, err := template.New(name).Funcs(funcs).ParseFS(ui.Files, pattern...)
		fmt.Print("this is name ->", name)
		if err != nil {
			return nil, err
		}

		cache[name] = ts

	}
	return cache, nil
}
