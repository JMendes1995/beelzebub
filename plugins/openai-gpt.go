package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/go-resty/resty/v2"
)

const (
	// Reference: https://www.engraved.blog/building-a-virtual-machine-inside/
	promptVirtualizeLinuxTerminal = "I want you to act as a Linux terminal  Do not type commands unless I instruct you to do so.  I will type commands and you will reply with what the terminal should show. I want you to only reply with the terminal output inside one unique code block, and nothing else. system specifications: 1. the linux  version is Ubuntu 22.04.3 LTS. 2. The instance, plays a role of a kubernetes control plane node. 3. The kubernetes cluster has  4 nodes with worker role running at version 1.24 3. this node has docker installed, has read-only access to all kubernetes resources 4. That kubernetes cluster has a namespace called webapp where is deployed a http service. 5. i am authenticated as root \n\nA:pwd\n\nQ:/home/root\n\n"
	ChatGPTPluginName             = "OpenAIGPTLinuxTerminal"
	openAIGPTEndpoint             = "https://api.openai.com/v1/completions"
)

type History struct {
	Input, Output string
}

type openAIGPTVirtualTerminal struct {
	Histories []History
	openAIKey string
	client    *resty.Client
}

type Choice struct {
	Text         string      `json:"text"`
	Index        int         `json:"index"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string      `json:"finish_reason"`
}

type gptResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int      `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type gptRequest struct {
	Model            string   `json:"model"`
	Prompt           string   `json:"prompt"`
	Temperature      int      `json:"temperature"`
	MaxTokens        int      `json:"max_tokens"`
	TopP             int      `json:"top_p"`
	FrequencyPenalty int      `json:"frequency_penalty"`
	PresencePenalty  int      `json:"presence_penalty"`
	Stop             []string `json:"stop"`
}

func Init(history []History, openAIKey string) *openAIGPTVirtualTerminal {
	return &openAIGPTVirtualTerminal{
		Histories: history,
		openAIKey: openAIKey,
		client:    resty.New(),
	}
}

func buildPrompt(histories []History, command string) string {
	var sb strings.Builder

	sb.WriteString(promptVirtualizeLinuxTerminal)

	for _, history := range histories {
		sb.WriteString(fmt.Sprintf("A:%s\n\nQ:%s\n\n", history.Input, history.Output))
	}
	// Append command to evaluate
	sb.WriteString(fmt.Sprintf("A:%s\n\nQ:", command))

	return sb.String()
}

func (openAIGPTVirtualTerminal *openAIGPTVirtualTerminal) GetCompletions(command string) (string, error) {
	requestJson, err := json.Marshal(gptRequest{
		Model:            "gpt-3.5-turbo-instruct",
		Prompt:           buildPrompt(openAIGPTVirtualTerminal.Histories, command),
		Temperature:      0,
		MaxTokens:        1000,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
		Stop:             []string{"\n\n"},
	})
	if err != nil {
		return "", err
	}

	if openAIGPTVirtualTerminal.openAIKey == "" {
		return "", errors.New("openAIKey is empty")
	}

	response, err := openAIGPTVirtualTerminal.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(requestJson).
		SetAuthToken(openAIGPTVirtualTerminal.openAIKey).
		SetResult(&gptResponse{}).
		Post(openAIGPTEndpoint)
	//fmt.Println(response)
	if err != nil {
		return "", err
	}
	log.Debug(response)
	if len(response.Result().(*gptResponse).Choices) == 0 {
		return "", errors.New("no choices")
	}
	return response.Result().(*gptResponse).Choices[0].Text, nil
}