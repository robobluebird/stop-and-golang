package main

import (
  "github.com/robobluebird/cards"
  "log"
  "io/ioutil"
  "net/http"
  "html/template"
  "strings"
  "regexp"
  "time"
  "github.com/gorilla/context"
  "github.com/gorilla/securecookie"
  "github.com/gorilla/sessions"
  "database/sql"
  _ "code.google.com/p/go-sqlite/go1/sqlite3"
  "github.com/coopernurse/gorp"
)

type Page struct {
  Title string
  Body []byte
}

func newCard(title string, body string) cards.CardBase {
    return cards.CardBase{
        Created: time.Now().UnixNano(),
        Title:   title,
        Body:    body,
    }
}

var valid_path = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var templates = template.Must(template.ParseFiles("templates/front.html", "templates/login.html", "templates/edit.html", "templates/view.html"))
var store = sessions.NewCookieStore([]byte(securecookie.GenerateRandomKey(32)))
var dbmap *gorp.DbMap

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

func sessionHandler(w http.ResponseWriter, r *http.Request, action string) {
  //username, password := r.FormValue("username"), r.FormValue("password")
  //get user, check password hash here
  
  session, _ := store.Get(r, "the-session")
  
  if action == "new" {
    session.Values["user_id"] = securecookie.GenerateRandomKey(32)
    session.Save(r, w)
    redirect_path, ok := context.GetOk(r, "requested_path")
    if !ok {
      redirect_path = "front"
    }
    http.Redirect(w, r, redirect_path.(string), http.StatusFound)
  } else if action == "destroy" {
    delete(session.Values, "user_id")
    http.Redirect(w, r, "/", http.StatusFound)
  } else {
    http.NotFound(w, r)
  }
}

func makeHandler(fn func (http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    m := valid_path.FindStringSubmatch(r.URL.Path)
    if m == nil {
      http.NotFound(w, r)
      return
    }
    
    session, _ := store.Get(r, "the-session")
    if session.Values["user_id"] == nil {
      context.Set(r, "requested_path", r.URL.Path)
      http.Redirect(w, r, "/login", http.StatusFound)
    }
    
    fn(w, r, m[2])
  }
}

func initDb() *gorp.DbMap {
  db, err := sql.Open("sqlite3", "card.db")
  checkErr(err, "sql.Open failed")
  dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
  
  dbmap.AddTableWithName(cards.CardBase{}, "cards").SetKeys(true, "Id")
  dbmap.AddTableWithName(cards.CardImage{}, "images").SetKeys(true, "Id")
  dbmap.AddTableWithName(cards.CardExtra{}, "extras").SetKeys(true, "Id")
  err = dbmap.CreateTablesIfNotExists()
  
  return dbmap
}

func checkErr(err error, msg string) {
  if err != nil {
    log.Fatalln(msg, err)
  }
}

func main() {
  dbmap = initDb()
  defer dbmap.Db.Close()
  http.HandleFunc("/", frontHandler)
  http.HandleFunc("/login", loginHandler)
  http.HandleFunc("/session/", makeHandler(sessionHandler))
  http.HandleFunc("/view/", makeHandler(viewHandler))
  http.HandleFunc("/edit/", makeHandler(editHandler))
  http.HandleFunc("/save/", makeHandler(saveHandler))
  http.ListenAndServe(":8080", context.ClearHandler(http.DefaultServeMux))
}
