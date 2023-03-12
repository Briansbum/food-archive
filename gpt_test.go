package main

import "testing"

var recipe1 = &Recipe{
	RecipeText: `
Serving Size: 2

Ingredients:
- 1 tbsp olive oil
- 1 small onion, diced
- 2 garlic cloves, minced
- 1/2 tsp smoked paprika
- 1/2 tsp cumin
- 1/2 tsp red pepper flakes
- 1 can of diced tomatoes (400g)
- 1/2 red bell pepper, sliced
- 1/2 yellow bell pepper, sliced
- 4 slices of halloumi cheese (120g)
- Salt and pepper to taste
- Fresh parsley for garnish

Instructions:
1. Heat the olive oil in a medium-sized pan over medium heat.
2. Add the diced onion and cook until translucent, about 5 minutes.
3. Add the minced garlic, smoked paprika, cumin, and red pepper flakes, and cook for 1-2 minutes.
4. Pour the can of diced tomatoes into the pan and stir well.
5. Add the sliced red and yellow bell peppers to the pan and stir to combine.
6. Bring the mixture to a simmer and let it cook for 10-15 minutes, until the peppers are softened.
7. Season with salt and pepper to taste.
8. Place the halloumi slices on top of the tomato-pepper mixture and cover the pan with a lid.
9. Cook for another 5-10 minutes, until the halloumi is melted and bubbly.
10. Garnish with fresh parsley and serve hot.

Serving/Presentation Suggestions:
- Serve the shakshuka in individual bowls, topped with extra parsley for color and flavor.
- Serve with a side of crusty bread for dipping and mopping up the sauce.

Modifications:
- For a spicier version, add more red pepper flakes or a diced jalapeno pepper.
- For a heartier version, add some cooked chickpeas or lentils to the mixture.
- For a vegetarian version, omit the halloumi and add some extra veggies like mushrooms or zucchini.					
`,
}

func Test_parseRecipeText(t *testing.T) {
	type args struct {
		recipe      *Recipe
		servingSize int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				recipe:      recipe1,
				servingSize: 2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseRecipeText(tt.args.recipe, tt.args.servingSize)
			if recipe1.Content.Servings != 2 {
				t.Errorf("expected serving size to be 2, got %d", recipe1.Content.Servings)
			}
			if len(recipe1.Content.Ingredients) != 12 {
				t.Errorf("expected 12 ingredients, got %d", len(recipe1.Content.Ingredients))
			}
			if len(recipe1.Content.MethodLines) != 10 {
				t.Errorf("expected 10 method lines, got %d", len(recipe1.Content.MethodLines))
			}
			if len(recipe1.Content.Suggestions) != 2 {
				t.Errorf("expected 2 suggestions, got %d", len(recipe1.Content.Suggestions))
			}
			if len(recipe1.Content.Modifications) != 3 {
				t.Errorf("expected 3 modifications, got %d", len(recipe1.Content.Modifications))
			}
		})
	}
}
