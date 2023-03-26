package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	templates *template.Template
)

type Recipe struct {
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
	db, err := bolt.Open("/data/boltdb", 0666, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	recipes := []*Recipe{}
	if err := loadRecipes(db, recipes); err != nil || len(recipes) == 0 {
		log.Printf("error loading recipes: %v\n", err)
		// log.Println("seeding recipes")
		// if err := seedRecipes(db); err != nil {
		// 	log.Fatal(err)
		// }
	}
	if err := runMigrations(recipes); err != nil {
		log.Fatal(err)
	}

	go func() {
		r := 0
		for {
			nr, err := flushRecipes(db, recipes, r)
			if err != nil {
				fmt.Println(fmt.Errorf("error flushing recipes: %w", err).Error())
			}
			r = nr
			time.Sleep(5 * time.Second)
		}
	}()

	t, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}
	templates = t

	mux := http.NewServeMux()

	registerRoutes(mux, recipes)

	fmt.Println("Listening on port 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}

func loadRecipes(db *bolt.DB, recipes []*Recipe) error {
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
			if len(recipes) > 0 {
				break
			}
		}
		fmt.Printf("loaded %d recipes from boltdb\n", len(recipes))
		return nil
	})
}

func flushRecipes(db *bolt.DB, recipes []*Recipe, lastRecipesSize int) (int, error) {
	r, err := json.Marshal(recipes)
	if err != nil {
		return lastRecipesSize, err
	}
	if len(r) == lastRecipesSize {
		return lastRecipesSize, nil // no change in recipes since last flush
	}
	return len(r), db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("recipes"))
		if err != nil {
			return err
		}
		t := time.Now().UnixNano() / int64(time.Millisecond)
		if err := b.Put([]byte(strconv.Itoa(int(t))), []byte(r)); err != nil {
			return err
		}
		fmt.Printf("flushed %d recipes to boltdb\n", len(recipes))
		return nil
	})
}

func seedRecipes(db *bolt.DB, recipes []*Recipe) error {
	f, err := os.Open("recipes_with_tags.json")
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&recipes); err != nil {
		return err
	}

	fmt.Printf("seeding %d recipes to boltdb\n", len(recipes))

	_, err = flushRecipes(db, recipes, 0)
	return err
}

func runMigrations(recipes []*Recipe) error {
	// if a recipe doesn't have a version, set it to 1
	for _, r := range recipes {
		if r.Version == 0 {
			r.Version = 1
		}
	}
	return nil
}