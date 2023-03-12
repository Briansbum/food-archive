package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/list", basicAuth(list))
	mux.HandleFunc("/recipe", basicAuth(recipe))
	mux.HandleFunc("/extract", basicAuth(extractRecipes))
	mux.HandleFunc("/create", basicAuth(create))
}

func list(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(res, "error: method not allowed")
		return
	}

	if err := templates.ExecuteTemplate(res, "list.html", recipes); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "error rendering list: %v", err)
		return
	}
}

func recipe(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(res, "error: method not allowed")
		return
	}

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

	regenerateParam := req.URL.Query().Get("regenerate")
	var regenerate bool
	if regenerateParam != "" {
		regenerate, err = strconv.ParseBool(regenerateParam)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(res, "error: invalid regenerate provided")
			return
		}
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
	if recipe.RecipeText == "" || regenerate {
		if err := generateRecipe(recipe, servingSizeInt); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(res, "error generating recipe: %v", err)
			return
		}
	}

	if err := templates.ExecuteTemplate(res, "recipe.html", recipe); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "error rendering recipe: %v", err)
		return
	}
}

func extractRecipes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "error: method not allowed")
		return
	}

	recipesJSON, err := json.Marshal(recipes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error marshalling recipes: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(recipesJSON)
}

func create(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if err := templates.ExecuteTemplate(w, "create.html", nil); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "error rendering create: %v", err)
			return
		}
		return
	case http.MethodPost:
		break
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "error: method not allowed")
		return
	}

	// parse form
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error parsing form: %v", err)
		return
	}

	// validate form
	recipeName := r.FormValue("name")
	if recipeName == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: no recipe name provided")
		return
	}

	recipeURL := r.FormValue("url")

	servingSizeValue := r.FormValue("serving_size")
	if servingSizeValue == "" {
		servingSizeValue = "2"
	}
	var servingSize int
	if i, err := strconv.Atoi(servingSizeValue); err != nil || i <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error: invalid serving size provided")
		return
	} else {
		servingSize = i
	}

	tagsValue := r.FormValue("tags")
	var tags []string
	if tagsValue != "" {
		tags = strings.Split(tagsValue, ",")
	}

	ingredientsValue := r.FormValue("ingredients")
	ingredients := map[string]*IngredientAmount{}
	scanner := bufio.NewScanner(strings.NewReader(ingredientsValue))
	for scanner.Scan() {
		t := scanner.Text()
		ingredient, ingredientAmount, err := parseIngredientLine(t)
		if err != nil {
			ingredients[ingredient] = nil
		} else {
			ingredients[ingredient] = &ingredientAmount
		}
	}

	method := r.FormValue("method")
	scanner = bufio.NewScanner(strings.NewReader(method))
	var methodLines []string
	for scanner.Scan() {
		methodLines = append(methodLines, scanner.Text())
	}

	suggestions := r.FormValue("suggestions")
	scanner = bufio.NewScanner(strings.NewReader(suggestions))
	var suggestionLines []string
	for scanner.Scan() {
		suggestionLines = append(suggestionLines, scanner.Text())
	}

	modifications := r.FormValue("modifications")
	scanner = bufio.NewScanner(strings.NewReader(modifications))
	var modificationLines []string
	for scanner.Scan() {
		modificationLines = append(modificationLines, scanner.Text())
	}

	// create recipe
	recipe := &Recipe{
		Name:      recipeName,
		Reference: recipeURL,
		Tags:      tags,
		Content: &RecipeContent{
			Servings:      servingSize,
			Ingredients:   ingredients,
			MethodLines:   methodLines,
			Suggestions:   suggestionLines,
			Modifications: modificationLines,
		},
	}

	if err := generateTags(recipe, false); err != nil {
		fmt.Printf("error generating tags: %v", err)
	}

	recipes = append(recipes, recipe)

	// redirect to recipe
	http.Redirect(w, r, fmt.Sprintf("/recipe?name=%s&serving_size=%d", url.QueryEscape(recipeName), servingSize), http.StatusFound)
}
