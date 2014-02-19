package main

import (
  "io/ioutil"
  "net/http"
  "html/template"
  "strings"
  "regexp"
  "errors"
  "github.com/gorilla/context"
  "github.com/gorilla/securecookie"
  "github.com/gorilla/sessions"
)

type Page struct {
  Title string
  Body []byte
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var templates = template.Must(template.ParseFiles("templates/front.html", "templates/login.html", "templates/edit.html", "templates/view.html"))
var store = sessions.NewCookieStore([]byte(securecookie.GenerateRandomKey(32)))

func (p *Page) save() error {
  filename := "pages/" + strings.ToLower(p.Title) + ".txt"
  return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
  filename := "pages/" + strings.ToLower(title) + ".txt"
  body, err := ioutil.ReadFile(filename)
  if err != nil {
    return nil, err
  }
  return &Page{Title: title, Body: body}, nil
}

func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
  m := validPath.FindStringSubmatch(r.URL.Path)
  if m == nil {
    http.NotFound(w, r)
    return "", errors.New("Invalid Page Title")
  }
  return m[2], nil
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
  err := templates.ExecuteTemplate(w, tmpl + ".html", p)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

func frontHandler(w http.ResponseWriter, r *http.Request) {
  renderTemplate(w, "front", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
  renderTemplate(w, "login", nil)
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
  p, err := loadPage(title)
  if err != nil {
    http.Redirect(w, r, "/edit/" + title, http.StatusFound)
    return
  }
  renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
  p, err := loadPage(title)
  if err != nil {
    p = &Page{Title: title}
  }
  renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
  body := r.FormValue("body")
  p := &Page{Title: title, Body: []byte(body)}
  err := p.save()
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  http.Redirect(w, r, "/view/" + title, http.StatusFound)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
  //username, password := r.FormValue("username"), r.FormValue("password")
  //get user, check password hash here
  
  session, _ := store.Get(r, "the-session")
  session.Values["user_id"] = securecookie.GenerateRandomKey(32)
  session.Save(r, w)
  http.Redirect(w, r, "/view/wat", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
  session, _ := store.Get(r, "the-session")
  delete(session.Values, "user_id")
  http.Redirect(w, r, "/", http.StatusFound)
}

func makeHandler(fn func (http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "the-session")
    if session.Values["user_id"] == nil {
      http.Redirect(w, r, "/login", http.StatusFound)
    }
    
    m := validPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
      http.NotFound(w, r)
      return
    }
    
    fn(w, r, m[2])
  }
}

func main() {
  http.HandleFunc("/", frontHandler)
  http.HandleFunc("/login", loginHandler)
  http.HandleFunc("/sessions/new", sessionHandler)
  http.HandleFunc("/view/", makeHandler(viewHandler))
  http.HandleFunc("/edit/", makeHandler(editHandler))
  http.HandleFunc("/save/", makeHandler(saveHandler))
  http.ListenAndServe(":8080", context.ClearHandler(http.DefaultServeMux))
}
