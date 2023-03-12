package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/PullRequestInc/go-gpt3"
)

var (
	client = gpt3.NewClient(os.Getenv("OPENAPI_KEY"), gpt3.WithDefaultEngine(gpt3.GPT3Dot5Turbo))
)

func generateTags(overrideTags bool) {
	for i := range recipes {
		resp, err := client.ChatCompletion(context.Background(), gpt3.ChatCompletionRequest{
			Messages: []gpt3.ChatCompletionRequestMessage{
				{
					Role:    "system",
					Content: "You are a data tagger for a global food entertainment brand. Your role is to read recipe titles/links and, using your exhaustive knowledge of food, provide ten tags for the recipe as a json string array. It can include cuisine, ingredients, cooking method, etc. For example, if you were given the recipe title “Chicken Tikka Masala”, you would return [“Indian”, “Chicken”, “Curry”]. Bias towrads ingredients making up the bulk of the tags, if the recipe is suitable for lunch then always include a lunch tag",
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

		tagsString := resp.Choices[0].Message.Content

		tags := []string{}
		err = json.Unmarshal([]byte(tagsString), &tags)
		if err != nil {
			fmt.Printf("error unmarshalling tags '%s': %+v\n", tagsString, err)
			continue
		}

		if overrideTags {
			recipes[i].Tags = tags
		} else {
			recipes[i].Tags = append(recipes[i].Tags, tags...)
		}
	}
}

func generateRecipe(recipe *Recipe, servingSize int) error {
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(60*time.Second))
	defer cancelFunc()

	resp, err := client.ChatCompletion(ctx, gpt3.ChatCompletionRequest{
		Messages: []gpt3.ChatCompletionRequestMessage{
			{
				Role:    "system",
				Content: "You are a personal chef with extensive experience in the home cooking space. You are tasked with creating a recipe for a new dish. You are given a title and a serving size. You must create a recipe that is suitable for the given serving size. The output will be comprised of the following sections: Serving Size, Ingredients, Instructions, serving/Presentation Suggestions, Modifications. Include line breaks in your recipe to separate the different sections/paragraphs/lines.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("%s %d", recipe.Name, servingSize),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error calling openai: %w", err)
	}

	fmt.Println(resp)
	recipe.RecipeText = resp.Choices[0].Message.Content

	return nil
}
