package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var (
	templates *template.Template
	dbFile    string = "recipes.db"
)

type Recipe struct {
	ID         int            `csv:"id" json:"id"`
	Version    int            `csv:"version" json:"version"`
	Name       string         `csv:"name" json:"name"`
	Reference  string         `csv:"reference" json:"reference"`
	Tags       []string       `csv:"tags" json:"tags"`
	RecipeText string         `csv:"recipe_text" json:"recipe_text"`
	Content    *RecipeContent `csv:"-" json:"content"`
}

type RecipeContent struct {
	Servings      int
	Ingredients   map[string]*IngredientAmount
	MethodLines   []string
	Suggestions   []string
	Modifications []string
}

type IngredientAmount struct {
	Amount string
	Unit   string
}

func main() {
	if filename, ok := os.LookupEnv("DB_FILE"); ok {
		dbFile = filename
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	if ok, err := checkDBSeeded(db); err != nil {
		log.Fatalf("unable to verify that the db is set up correctly, got err: %+v", err)
	} else if !ok {
		if err := seedRecipes(db); err != nil {
			log.Fatalf("unable to seed recipes, got err: %+v\n", err)
		}
	}

	t, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}
	templates = t

	mux := http.NewServeMux()

	registerRoutes(mux, db)

	fmt.Println("Listening on port 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
