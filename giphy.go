package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type GifClient struct {
	Key     string
	PerPage int
	client  *http.Client
}

type GifResult struct {
	Gif GifData `json:"data"`
}

type SearchResult struct {
	Gifs []GifData `json:"data"`
}

type GifData struct {
	Id     string  `json:"id"`
	URL    string  `json:"url"`
	Images Images  `json:"images"`
}

type Images struct {
	FixedWidthSmallStill ImageData `json:"fixed_width_small_still"`
	Downsized ImageData `json:"downsized"`
}

type ImageData struct {
	URL string `json:"url"`
}

type Gif struct {
	Id           string `json:"id"`
	URL          string `json:"url"`
	StillURL     string `json:"still_url"`
	DownsizedURL string `json:"downsized_url"`
}

type Tag struct {
	Favorite string `json:"favorite"`
	Tag      string `json:"tag"`
}

func (c *GifClient) Get(id string) (*Gif, error) {
	url := "https://api.giphy.com/v1/gifs/" + id
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("Couldnt create request: %s", reqErr)
	}
	q := req.URL.Query()
	q.Add("api_key", c.Key)
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	var search GifResult
	htmlData, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, readErr
	}
	// FIXME. Use decoder.
	//err = json.NewDecoder(resp.Body).Decode(&search)
	err = json.Unmarshal(htmlData, &search)
	if err != nil {
		return nil, err
	}
	gif := search.Gif
	return &Gif{
		Id: gif.Id,
		URL: gif.URL,
		StillURL: gif.Images.FixedWidthSmallStill.URL,
		DownsizedURL: gif.Images.Downsized.URL}, nil
}

func (c *GifClient) Search(query string, page int) ([]Gif, error) {
	url := "https://api.giphy.com/v1/gifs/search"
	req, reqErr := http.NewRequest("GET", url, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("Couldnt create request: %s", reqErr)
	}
	q := req.URL.Query()
	q.Add("api_key", c.Key)
	q.Add("rating", "g")
	q.Add("q", query)
	q.Add("limit", strconv.Itoa(c.PerPage))
	if page < 1 {
		page = 1
	}
	q.Add("offset", strconv.Itoa((page - 1) * c.PerPage))
	req.URL.RawQuery = q.Encode()

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
	//fmt.Printf("entire body: %+v\n", string(htmlData[:]))
	//err = json.NewDecoder(resp.Body).Decode(&search)
	err = json.Unmarshal(htmlData, &search)
	if err != nil {
		return nil, err
	}
	results := make([]Gif, 0, len(search.Gifs))
	for _, gif := range search.Gifs {
		result := Gif{Id: gif.Id, URL: gif.URL, StillURL: gif.Images.FixedWidthSmallStill.URL, DownsizedURL: gif.Images.Downsized.URL}
		results = append(results, result)
	}

	return results, nil
}

func NewGifClient(key string, perPage int) *GifClient{
	if perPage < 1 {
		perPage = 1
	}
	return &GifClient{Key: key, client: http.DefaultClient, PerPage: perPage}
}
