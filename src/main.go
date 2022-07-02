package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	editPath = "/edit/"
	savePath = "/save/"
	viewPath = "/view/"
	listPath = "/"
)

const (
	editTemplate = "edit.html"
	viewTemplate = "view.html"
	listTemplate = "list.html"
)

const (
	dataDir = "data"
)

var (
	templateDir = "tmpl"
	templates   = template.Must(template.ParseFiles(
		filepath.Join(templateDir, editTemplate),
		filepath.Join(templateDir, viewTemplate),
		filepath.Join(templateDir, listTemplate),
	))
)

var (
	titlePathExp = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
	pageLinkExp  = regexp.MustCompile(`\[([a-zA-Z0-9]+)\]`)
)

type Page struct {
	Title string
	Body  []byte
	Index []string
}

func (p *Page) Save() error {
	filename := p.Title + ".txt"
	return os.WriteFile(filepath.Join(dataDir, filename), p.Body, 0600)
}

func (p *Page) Html() template.HTML {
	html := []byte(pageLinkExp.ReplaceAllString(string(p.Body), `<a href="/view/$1">$1</a>`))
	return template.HTML(html)
}

func loadIndex() ([]string, error) {
	l, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}
	r := make([]string, 0)
	for _, v := range l {
		if t, n := v.Type(), v.Name(); t.IsRegular() && strings.HasSuffix(n, ".txt") {
			r = append(r, n[:len(n)-len(".txt")])
		}
	}
	return r, nil
}

func loadPage(title string) (*Page, error) {
	filename := title + ".txt"
	body, err := os.ReadFile(filepath.Join(dataDir, filename))
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func titleHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := titlePathExp.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, editTemplate, p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.Save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, viewPath+title, http.StatusFound)
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, editPath+title, http.StatusFound)
		return
	}
	renderTemplate(w, viewTemplate, p)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	i, err := loadIndex()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderTemplate(w, listTemplate, &Page{Index: i})
}

func main() {
	http.HandleFunc(editPath, titleHandler(editHandler))
	http.HandleFunc(savePath, titleHandler(saveHandler))
	http.HandleFunc(viewPath, titleHandler(viewHandler))
	http.HandleFunc(listPath, listHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
