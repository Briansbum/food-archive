# food-archive

This program runs the food archive helper site

## Running in dev

Get an OPENAI api key from https://platform.openai.com/account/api-keys

export it into your shell as OPENAI_KEY

generate a username/password with `htpasswd -n` and export the password section as `USER_{username}_PASSWORD`

run `go run .`
