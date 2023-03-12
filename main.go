package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/pat"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	bolt "go.etcd.io/bbolt"
)

var (
	recipes       = []Recipe{}
	dataPath      = "/data/boltdb"
	allowedEmails = map[string]bool{
		"freestone.alex@gmail.com": true,
		// "cassidy.hall94@gmail.com",
	}
)

type Recipe struct {
	Name      string   `csv:"name" json:"name"`
	Reference string   `csv:"reference" json:"reference"`
	Tags      []string `csv:"tags" json:"tags"`
}

func main() {
	db, err := bolt.Open(dataPath, 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := loadRecipes(db); err != nil || len(recipes) == 0 {
		if err2 := seedRecipes(db); err != nil {
			log.Fatal(err, err2)
		}
	}

	go func() {
		for {
			if err := flushRecipes(db); err != nil {
				fmt.Println(fmt.Errorf("error flushing recipes: %w", err).Error())
			}
			time.Sleep(5 * time.Second)
		}
	}()

	templates, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}

	goth.UseProviders(
		github.New(os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"), "https://muddy-leaf-8313.fly.dev/auth/github/callback", "user:email"),
	)

	p := pat.New()
	p.Get("/auth/{provider}/callback", func(w http.ResponseWriter, r *http.Request) {
		if isAuthorized(w, r, true) {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "not authorized")
		}
	})

	p.Get("/auth/{provider}", func(w http.ResponseWriter, r *http.Request) {
		// try to get the user without re-authenticating
		if isAuthorized(w, r, true) {
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		} else {
			gothic.BeginAuthHandler(w, r)
		}
	})

	p.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if !isAuthorized(w, r, true) {
			http.Redirect(w, r, "/auth/github", http.StatusTemporaryRedirect)
			return
		}

		if err := renderIndex(w, templates); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error rendering index: %v", err)
			return
		}
	})

	fmt.Println("Listening on port 8080")
	if err := http.ListenAndServe(":8080", p); err != nil {
		panic(err)
	}
}

func isAuthorized(w http.ResponseWriter, r *http.Request, retry bool) bool {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		if retry && strings.Contains(r.URL.Path, "auth") {
			gothic.BeginAuthHandler(w, r)
			return isAuthorized(w, r, false)
		}
		return false
	}
	return user.Email != "" && allowedEmails[user.Email]
}

func renderIndex(w http.ResponseWriter, templates *template.Template) error {
	if err := templates.ExecuteTemplate(w, "index.html", recipes); err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}
	return nil
}

func loadRecipes(db *bolt.DB) error {
	return db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("recipes"))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			if err := json.Unmarshal(v, &recipes); err != nil {
				return err
			}
		}
		return nil
	})
}

func flushRecipes(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("recipes"))
		if err != nil {
			return err
		}
		t := time.Now().UnixNano() / int64(time.Millisecond)
		r, err := json.Marshal(recipes)
		if err != nil {
			return err
		}
		if err := b.Put([]byte(strconv.Itoa(int(t))), []byte(r)); err != nil {
			return err
		}
		return nil
	})
}

func seedRecipes(db *bolt.DB) error {
	f, err := os.Open("recipes_with_tags.json")
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&recipes); err != nil {
		return err
	}

	return flushRecipes(db)
}
