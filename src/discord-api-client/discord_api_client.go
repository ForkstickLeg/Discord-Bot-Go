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
	tempMap := make(map[string]string)
	for i := 0; i < len(key); i++ {
		tempMap[key[i]] = value[i]
	}
	ac.header = tempMap
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
	req, err := http.NewRequest("POST", ac.ApiUrl, strings.NewReader(string(ac.body)))
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
	req, err := http.NewRequest("GET", ac.ApiUrl, nil)
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

func (ac *ApiCall) MakePatchCall() []byte {
	req, err := http.NewRequest("PATCH", ac.ApiUrl, strings.NewReader(string(ac.body)))
	if err != nil {
		fmt.Println("Error making PATCH request")
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
