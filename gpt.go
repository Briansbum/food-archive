package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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
