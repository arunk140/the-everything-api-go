package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

const basePrompt = `Create a response document with content that matches the following URL path: 
'{{URL_PATH}}'

The first line is the Content-Type of the response.
The following lines is the returned data.
In case of a html response, add relative href links with to related topics. Also add basic CSS to make it look good.
{{OPTIONAL_DATA}}

Content-Type: {{CONTENT_TYPE}}
`

func main() {
	// load .env file
	godotenv.Load()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY is not set")
		return
	}

	client := openai.NewClient(apiKey)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]
		var prompt string
		var contentType string
		if r.Method == http.MethodPost {
			formData, err := json.Marshal(r.Form)
			if err != nil {
				fmt.Println("Error marshaling form data:", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			prompt = basePrompt
			prompt = replaceAll(prompt, "{{OPTIONAL_DATA}}", fmt.Sprintf("form data: %s", string(formData)))
		} else {
			prompt = basePrompt
			prompt = replaceAll(prompt, "{{OPTIONAL_DATA}}", "")
		}
		prompt = replaceAll(prompt, "{{URL_PATH}}", path)
		if strings.Contains(path, "api/") {
			contentType = "application/json"
		} else {
			contentType = "text/html"
		}
		prompt = replaceAll(prompt, "{{CONTENT_TYPE}}", contentType)

		if path == "favicon.ico" {
			http.ServeFile(w, r, "./favicon.ico")
			return
		}
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: "You are a HTTP Server, the user will provide a URL path and instructions and you strictly will only return a response document.",
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					},
				},
			},
		)
		if err != nil {
			fmt.Println("Error creating OpenAI completion:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		aiData := resp.Choices[0].Message.Content

		responseData := aiData
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseData))
	})

	fmt.Println("Listening on port 8080...")
	http.ListenAndServe(":8080", nil)
}

func replaceAll(s, old, new string) string {
	for {
		i := strings.Index(s, old)
		if i == -1 {
			return s
		}
		s = s[:i] + new + s[i+len(old):]
	}
}