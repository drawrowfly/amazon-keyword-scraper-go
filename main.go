package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Suggestion struct {
	SuggType       string `json:"suggType"`
	Value          string `json:"value"`
	RefTag         string `json:"refTag"`
	StrategyId     string `json:"strategyId"`
	Ghost          bool   `json:"ghost"`
	Help           bool   `json:"help"`
	Fallback       bool   `json:"fallback"`
	SpellCorrected bool   `json:"spellCorrected"`
	BlackListed    bool   `json:"blackListed"`
	XcatOnly       bool   `json:"xcatOnly"`
}
type KeywordSuggestions struct {
	Alias             string       `json:"alias"`
	Prefix            string       `json:"prefix"`
	Suffix            string       `json:"suffix"`
	Suggestions       []Suggestion `json:"suggestions"`
	SuggestionTitleId string       `json:"suggestionTitleId"`
	ResponseId        string       `json:"responseId"`
	Shuffled          bool         `json:"shuffled"`
}

func main() {

	if len(os.Args) == 1 {
		log.Fatal("Keyword is required")
	}
	keyword := os.Args[1]

	k := make(chan string)
	list := make(chan []string)

	go grabKeyWords(k, list, true)

	k <- keyword

	readValues(list)

}

func readValues(list chan []string) {
	for {
		val, ok := <-list
		if ok == false {
			break
		} else {
			fmt.Println(val)
		}
	}
}

func grabKeyWords(k chan string, c chan []string, more bool) {
	client := http.Client{}

	req, _ := http.NewRequest("GET", "https://completion.amazon.com/api/2017/suggestions", nil)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.117 Safari/537.36")
	req.Header.Set("Origin", "https://www.amazon.com")
	req.Header.Set("Referer", "https://www.amazon.com/")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,ru;q=0.8")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")

	q := req.URL.Query()
	q.Add("api_key", "key_from_environment_or_flag")
	q.Add("page-type", "Gateway")
	q.Add("lop", "en_US")
	q.Add("site-variant", "desktop")
	q.Add("client-info", "amazon-search-ui")
	q.Add("mid", "ATVPDKIKX0DER")
	q.Add("alias", "aps")
	q.Add("b2b", "0")
	q.Add("fresh", "0")
	q.Add("ks", "80")
	q.Add("prefix", <-k)
	q.Add("event", "onKeyPress")
	q.Add("limit", "11")
	q.Add("fb", "1")
	q.Add("suggestion-type", "KEYWORD")
	q.Add("_", string(time.Now().UnixNano()/int64(time.Millisecond)))
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
	}

	var reader io.ReadCloser
	reader, _ = gzip.NewReader(resp.Body)
	defer reader.Close()

	var result KeywordSuggestions

	json.NewDecoder(reader).Decode(&result)

	var list []string
	for _, item := range result.Suggestions {
		list = append(list, item.Value)
	}
	c <- list

	if !more {
		close(c)
	}
}
