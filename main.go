package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {
	// read and decode recipes_with_tags.json
	f, err := os.Open("recipes_with_tags.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var recipes []Recipe
	if err := json.NewDecoder(f).Decode(&recipes); err != nil {
		log.Fatal(err)
	}

	templates, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := templates.ExecuteTemplate(w, "index.html", recipes); err != nil {
			fmt.Printf("error executing template: %v\n", err)
			return
		}
	})

	htmlFile, err := os.Create("index.html")
	if err != nil {
		log.Fatal(err)
	}
	if err := templates.ExecuteTemplate(htmlFile, "index.html", recipes); err != nil {
		fmt.Printf("error executing template: %v\n", err)
		return
	}

	fmt.Println("Listening on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
