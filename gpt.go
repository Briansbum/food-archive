package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
				Role: "system",
				Content: `
You are a personal chef with extensive experience in the home cooking space.
You are tasked with creating a recipe for a new dish.
You are given a title and a serving size. 
You must create a recipe that is suitable for the given serving size.
The output will be comprised of the following sections: Serving Size, Ingredients, Instructions, serving/Presentation Suggestions, Modifications.
Include line breaks in your recipe to separate the different sections/paragraphs/lines.
Weight and Volume units are in metric. Ingredients always have a name prefixed by a number and a unit. The unit is always singular. The number is always a whole number or a decimal to a maximum of 2 decimal places. The number and unit are separated by a space. The name is always lowercase. The number and unit are always lowercase. The number and unit are always separated by a space. If the ingredient has a quantity the name is always separated from the number and unit by an exclamation mark.
Ingredient examples: "1 cup flour" is "250g flour", "1tsp salt" is "1tsp salt" (no change), "1/2 cup sugar" is "125g sugar", "1/2 tsp salt" is "1/2 tsp salt" (no change).
				`,
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

	recipe.RecipeText = resp.Choices[0].Message.Content
	parseRecipeText(recipe)

	return nil
}

func parseRecipeText(recipe *Recipe) {
	recipeContent := RecipeContent{
		Ingredients:   map[string]*IngredientAmount{},
		MethodLines:   []string{},
		Suggestions:   []string{},
		Modifications: []string{},
	}

	scanner := bufio.NewScanner(strings.NewReader(recipe.RecipeText))
	var section string
	for scanner.Scan() {
		if scanner.Text() == "" {
			continue
		}
		if scanner.Text() == "Ingredients" {
			section = "Ingredients"
			continue
		}
		if scanner.Text() == "Instructions" {
			section = "Instructions"
			continue
		}
		if scanner.Text() == "Serving/Presentation Suggestions" {
			section = "Serving/Presentation Suggestions"
			continue
		}
		if scanner.Text() == "Modifications" {
			section = "Modifications"
			continue
		}

		switch section {
		case "Ingredients":
			ingredient, ingredientAmount, err := parseIngredientLine(scanner.Text())
			if err != nil {
				fmt.Println("ingredient line may be malformed:", err)
				recipeContent.Ingredients[scanner.Text()] = nil
			}
			recipeContent.Ingredients[ingredient] = &ingredientAmount
		case "Instructions":
			recipeContent.MethodLines = append(recipeContent.MethodLines, scanner.Text())
		case "Serving/Presentation Suggestions":
			recipeContent.Suggestions = append(recipeContent.Suggestions, scanner.Text())
		case "Modifications":
			recipeContent.Modifications = append(recipeContent.Modifications, scanner.Text())
		default:
		}
	}

	recipe.Content = &recipeContent
}

func parseIngredientLine(line string) (string, IngredientAmount, error) {
	splitLine := strings.Split(line, "!")
	if len(splitLine) < 2 {
		return line, IngredientAmount{}, fmt.Errorf("could not find unit in line: %s", line)
	}
	amountSplit := strings.Split(splitLine[0], " ")
	ingredientAmount := IngredientAmount{
		Amount: amountSplit[0],
		Unit:   amountSplit[1],
	}
	return splitLine[1], ingredientAmount, nil
}
