package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/PullRequestInc/go-gpt3"
)

type Recipe struct {
	Name      string   `csv:"name" json:"name"`
	Reference string   `csv:"reference" json:"reference"`
	Tags      []string `csv:"tags" json:"tags"`
}

func generate() {
	// Open and decode recipes.json
	f, err := os.Open("recipes.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	var recipes []Recipe
	if err := json.NewDecoder(f).Decode(&recipes); err != nil {
		log.Fatal(err)
	}

	client := gpt3.NewClient("APIKEY", gpt3.WithDefaultEngine(gpt3.GPT3Dot5Turbo))

	for i := range recipes {
		resp, err := client.ChatCompletion(context.Background(), gpt3.ChatCompletionRequest{
			Messages: []gpt3.ChatCompletionRequestMessage{
				{
					Role:    "system",
					Content: "You are a data tagger for a global food entertainment brand. Your role is to read recipe titles/links and, using your exhaustive knowledge of food, provide ten tags for the recipe as a json string array. It can include cuisine, ingredients, cooking method, etc. For example, if you were given the recipe title “Chicken Tikka Masala”, you would return [“Indian”, “Chicken”, “Curry”]. Bias towrads ingredients making up the bulk of the tags",
				},
				{
					Role:    "user",
					Content: recipes[i].Name,
				},
			},
		})
		if err != nil {
			fmt.Println("error calling openai:", err)
		}

		fmt.Println(recipes[i].Name)
		fmt.Printf("resp: %+v\n", resp)

		tagsString := resp.Choices[0].Message.Content

		tags := []string{}
		err = json.Unmarshal([]byte(tagsString), &tags)
		if err != nil {
			fmt.Printf("error unmarshalling tags '%s': %+v\n", tagsString, err)
			continue
		}

		recipes[i].Tags = tags
	}

	// Encode recipes into recipes_with_tags.json
	f, err = os.Create("recipes_with_tags.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(recipes); err != nil {
		log.Fatal(err)
	}
}
