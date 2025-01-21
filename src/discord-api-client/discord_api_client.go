package discordapiclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ApiCall struct {
	ApiUrl string
	header map[string]string
	body   []byte
}

func NewApiCall(url string) *ApiCall {
	return &ApiCall{
		ApiUrl: url,
		header: nil,
		body:   nil,
	}
}

func (ac *ApiCall) AddHeader(key []string, value []string) {
	for i := 0; i < len(key); i++ {
		ac.header[key[i]] = value[i]
	}
}

func (ac *ApiCall) AddBody(data interface{}) {
	body, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Error marshalling data")
		return
	}
	ac.body = body
}

func (ac *ApiCall) MakePostCall() []byte {
	jsonBody, err := json.Marshal(ac.body)
	if err != nil {
		fmt.Println("Error marshalling body")
		return nil
	}
	req, err := http.NewRequest("POST", ac.ApiUrl, strings.NewReader(string(jsonBody)))
	if err != nil {
		fmt.Println("Error making POST request")
		return nil
	}

	for key, value := range ac.header {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making call")
		return nil
	}
	defer response.Body.Close()

	output, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response")
		return nil
	}
	return output
}

func (ac *ApiCall) MakeGetCall() []byte {
	req, err := http.NewRequest("POST", ac.ApiUrl, nil)
	if err != nil {
		fmt.Println("Error making POST request")
		return nil
	}

	for key, value := range ac.header {
		req.Header.Add(key, value)
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making call")
		return nil
	}
	defer response.Body.Close()

	output, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response")
		return nil
	}
	return output
}
