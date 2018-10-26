package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type GifClient struct {
	Key    string
	client *http.Client
}

type SearchResult struct {
	Gifs []GifData `json:"data"`
}

type GifData struct {
	Id       string `json:"id"`
	EmbedURL string `json:"embed_url"`
}

func (c *GifClient) Search(query string) ([]string, error) {
	url := "https://api.giphy.com/v1/gifs/search"
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("Couldnt create request: %s", reqErr)
	}
	q := req.URL.Query()
	q.Add("api_key", c.Key)
	q.Add("rating", "g")
	q.Add("q", query)
	req.URL.RawQuery = q.Encode()

	fmt.Printf("REQ: %+v\n", req)

	resp, err := c.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	var result SearchResult
	htmlData, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	fmt.Printf("entire body: %+v\n", string(htmlData[:]))
	//err = json.NewDecoder(htmlData).Decode(&result)
	err = json.Unmarshal(htmlData, &result)
	if err != nil {
		return nil, err
	}
	fmt.Printf("entire result: %+v\n", result)
	fmt.Printf("entire result.Gifs: %+v\n", result.Gifs)
	results := make([]string, 0, 20)
	for _, gif := range result.Gifs {
		fmt.Printf("result gif: %+v\n", gif)
		results = append(results, gif.EmbedURL)
	}
	return results, nil
}

func NewGifClient(key string) *GifClient{
	return &GifClient{Key: key, client: http.DefaultClient}
}
