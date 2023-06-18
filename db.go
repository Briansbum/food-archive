package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func insertRecipeVersion(db *sql.DB, recipe *Recipe) (*Recipe, error) {
	storedIDForParent := recipe.ID
	recipe.ID = 0

	insertPrep, err := db.Prepare("INSERT INTO recipes(parent_id, version, name, reference, recipe_data) values(?,?,?,?,?)")
	if err != nil {
		return nil, fmt.Errorf("unable to prepare for recipe insertion: %w", err)
	}

	recipe_data, err := json.Marshal(recipe)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal recipe as json for insertion: %w", err)
	}

	if _, err := insertPrep.Exec(storedIDForParent, recipe.Version+1, recipe.Name, recipe.Reference, recipe_data); err != nil {
		return nil, fmt.Errorf("unable to insert new recipe version: %w", err)
	}

	// read recipe back on parent_id
	readbackPrep, err := db.Prepare("SELECT id, recipe_data FROM recipes WHERE parent_id = ?")
	if err != nil {
		return nil, fmt.Errorf("unable to prepare for readback, your new recipe can be found in /list")
	}

	var (
		idN         sql.NullInt16
		recipeDataN sql.NullString
	)
	if err := readbackPrep.QueryRow(storedIDForParent).Scan(&idN, &recipeDataN); err != nil {
		return nil, fmt.Errorf("unable to scan row: %w", err)
	}

	var (
		id              int
		new_recipe_data string
	)
	if ok := idN.Valid; ok {
		id = int(idN.Int16)
	} else {
		return nil, fmt.Errorf("unable to find recipe")
	}
	if ok := recipeDataN.Valid; ok {
		new_recipe_data = recipeDataN.String
	}

	newRecipe := &Recipe{}
	if err := json.Unmarshal([]byte(new_recipe_data), newRecipe); err != nil {
		return nil, fmt.Errorf("unable to unmarshal recipe we just wrote: %w", err)
	}

	newRecipe.ID = id

	return newRecipe, nil
}

func getAllRecipeMeta(db *sql.DB) ([]*Recipe, error) {
	prep, err := db.Prepare("SELECT id, parent_id, version, name, reference FROM recipes")
	if err != nil {
		return nil, err
	}

	recipesMap := map[int]*Recipe{}

	rows, err := prep.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			idN        sql.NullInt16
			parentIDn  sql.NullInt16
			versionN   sql.NullInt16
			nameN      sql.NullString
			referenceN sql.NullString
		)
		if err := rows.Scan(&idN, &parentIDn, &versionN, &nameN, &referenceN); err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}

		var (
			id        int
			parentID  int
			version   int
			name      string
			reference string
		)

		if ok := idN.Valid; ok {
			id = int(idN.Int16)
		} else {
			return nil, fmt.Errorf("id must exist, db may be corrupt?")
		}
		if ok := parentIDn.Valid; ok {
			parentID = int(parentIDn.Int16)
		}
		if ok := versionN.Valid; ok {
			version = int(versionN.Int16)
		}
		if ok := nameN.Valid; ok {
			name = nameN.String
		}
		if ok := referenceN.Valid; ok {
			reference = referenceN.String
		}

		recipe := &Recipe{
			ID:        id,
			Version:   version,
			Name:      name,
			Reference: reference,
		}
		if _, ok := recipesMap[parentID]; !ok {
			delete(recipesMap, parentID)
		}
		recipesMap[id] = recipe
	}

	recipes := []*Recipe{}
	for _, r := range recipesMap {
		recipes = append(recipes, r)
	}

	return recipes, nil
}

// getAllRecipes uses json unmarshalling to get every current version of every
// recipe in the db, prefer using getAllRecipeMeta and fetch only what you need
func getAllRecipes(db *sql.DB) ([]*Recipe, error) {
	prep, err := db.Prepare("SELECT id, parent_id, recipe_data FROM recipes")
	if err != nil {
		return nil, err
	}

	recipesMap := map[int]*Recipe{}

	rows, err := prep.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		recipe_data := []byte{}
		id, parentID := 0, 0
		if err := rows.Scan(&id, &parentID, &recipe_data); err != nil {
			return nil, err
		}

		recipe := &Recipe{
			ID: id,
		}
		if err := json.Unmarshal(recipe_data, recipe); err != nil {
			return nil, err
		}

		if _, ok := recipesMap[parentID]; !ok {
			delete(recipesMap, parentID)
		}
		recipesMap[id] = recipe
	}

	recipes := []*Recipe{}

	for _, r := range recipesMap {
		recipes = append(recipes, r)
	}

	return recipes, nil
}

func getRecipeByID(db *sql.DB, id int) (*Recipe, error) {
	prep, err := db.Prepare("SELECT recipe_data FROM recipes WHERE id = ?")
	if err != nil {
		return nil, fmt.Errorf("unable to prepare for getting single recipe: %w", err)
	}

	recipe_data := []byte{}
	if err := prep.QueryRow(id).Scan(&recipe_data); err != nil {
		return nil, fmt.Errorf("unable to scan recipe_data: %w", err)
	}

	recipe := &Recipe{}
	if err := json.Unmarshal(recipe_data, recipe); err != nil {
		return nil, fmt.Errorf("unable to unmarshal recipe_data: %w\njson string: %s", err, string(recipe_data))
	}

	return recipe, nil
}

func checkDBSeeded(db *sql.DB) (bool, error) {
	count := 0
	err := db.QueryRow("SELECT COUNT(*) FROM recipes").Scan(&count)
	if err != nil {
		if strings.Contains(err.Error(), "no such table: recipes") {
			return false, nil
		}
		return false, err
	}

	if count == 0 {
		return false, nil
	}

	return true, nil
}

func seedRecipes(db *sql.DB) error {
	seedStmt := `
CREATE TABLE IF NOT EXISTS recipes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER,
    version INTEGER,
    name TEXT,
    reference TEXT,
    recipe_data TEXT
);
    `
	_, err := db.Exec(seedStmt)
	if err != nil {
		return fmt.Errorf("unable to create recipes table: %w", err)
	}

	seedFile, err := os.Open("recipes_with_tags.json")
	if err != nil {
		return fmt.Errorf("unable to open seed file recipes_with_tags.json: %w", err)
	}

	fileBytes, err := io.ReadAll(seedFile)
	if err != nil {
		return fmt.Errorf("unable to read seed file contents: %w", err)
	}

	recipes := []*Recipe{}

	if err := json.Unmarshal(fileBytes, &recipes); err != nil {
		return fmt.Errorf("unable to unmarshal seed file contents into []*Recipe: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error beginning db transaction for seed: %w", err)
	}

	stmt, err := tx.Prepare("INSERT INTO recipes(version, name, reference, recipe_data) values(?,?,?,?)")
	if err != nil {
		return fmt.Errorf("unable to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	for i, r := range recipes {
		r := r

		r.ID = i + 1

		b, err := json.Marshal(r)
		if err != nil {
			return fmt.Errorf("unable to convert recipe back into json: %w", err)
		}

		if _, err := stmt.Exec(r.Version, r.Name, r.Reference, string(b)); err != nil {
			return fmt.Errorf("unable to execute seed insert: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("unable to commit seed transaction: %w", err)
	}

	return nil
}
