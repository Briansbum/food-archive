package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/list", basicAuth(list))
	mux.HandleFunc("/recipe", basicAuth(recipe))
}

func list(res http.ResponseWriter, req *http.Request) {
	if err := templates.ExecuteTemplate(res, "list.html", recipes); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "error rendering list: %v", err)
		return
	}
}

func recipe(res http.ResponseWriter, req *http.Request) {
	// parse recipe name and serving size from url
	recipeName := req.URL.Query().Get("name")
	if recipeName == "" {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "error: no recipe name provided")
		return
	}
	servingSize := req.URL.Query().Get("serving_size")
	if servingSize == "" {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "error: no serving size provided")
		return
	}
	servingSizeInt, err := strconv.Atoi(servingSize)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "error: invalid serving size provided")
		return
	}

	// find recipe
	var recipe *Recipe
	for _, r := range recipes {
		if r.Name == recipeName {
			recipe = r
			break
		}
	}
	if recipe == nil {
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "error: recipe not found")
		return
	}

	// generate recipe
	if err := generateRecipe(recipe, servingSizeInt); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "error generating recipe: %v", err)
		return
	}

	if err := templates.ExecuteTemplate(res, "recipe.html", recipe); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "error rendering recipe: %v", err)
		return
	}
}
