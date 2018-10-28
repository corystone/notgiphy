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
	Id       string  `json:"id"`
	EmbedURL string  `json:"embed_url"`
	Images   Images `json:"images"`
}

type Images struct {
	FixedWidthSmallStill ImageData `json:"fixed_width_small_still"`
	Downsized ImageData `json:"downsized"`
}

type ImageData struct {
	URL string `json:"url"`
}

type GifSearchResult struct {
	Id       string `json:"id"`
	EmbedURL string `json:"embed_url"`
	StillURL string `json:"still_url"`
	DownsizedURL string `json:"downsized_url"`
}

func (c *GifClient) Search(query string) ([]GifSearchResult, error) {
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

	var search SearchResult
	htmlData, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	fmt.Printf("entire body: %+v\n", string(htmlData[:]))
	//err = json.NewDecoder(htmlData).Decode(&search)
	err = json.Unmarshal(htmlData, &search)
	if err != nil {
		return nil, err
	}
	results := make([]GifSearchResult, 0, len(search.Gifs))
	for _, gif := range search.Gifs {
		result := GifSearchResult{Id: gif.Id, EmbedURL: gif.EmbedURL, StillURL: gif.Images.FixedWidthSmallStill.URL, DownsizedURL: gif.Images.Downsized.URL}
		results = append(results, result)
	}

	return results, nil
}

func NewGifClient(key string) *GifClient{
	return &GifClient{Key: key, client: http.DefaultClient}
}
