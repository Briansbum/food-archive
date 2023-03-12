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
	recipes   = []*Recipe{}
	dataPath  = "/data/boltdb"
	templates *template.Template
)

type Recipe struct {
	Name       string   `csv:"name" json:"name"`
	Reference  string   `csv:"reference" json:"reference"`
	Tags       []string `csv:"tags" json:"tags"`
	RecipeText string   `csv:"recipe_text" json:"recipe_text"`
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

	t, err := template.ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}
	templates = t

	mux := http.NewServeMux()

	registerRoutes(mux)

	fmt.Println("Listening on port 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
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
		fmt.Printf("loaded %d recipes from boltdb", len(recipes))
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
		fmt.Printf("flushed %d recipes to boltdb", len(recipes))
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

	fmt.Printf("seeding %d recipes to boltdb", len(recipes))

	return flushRecipes(db)
}
