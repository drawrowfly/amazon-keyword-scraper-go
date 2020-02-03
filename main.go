package main

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
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

type Keyword struct {
	Keyword string
}

func DoWork() {
	time.Sleep(500 * time.Millisecond)
}

func main() {

	concurency := flag.Int("concurency", 2, "the number of goroutines that are allowed to run concurrently")
	limit := flag.Int("limit", 100, "number of keywords to collect")
	keywordToUse := flag.String("keyword", "iphone", "keyword to use")
	flag.Parse()

	concurrentGoroutines := make(chan struct{}, *concurency)
	var wg sync.WaitGroup

	keyword := Keyword{*keywordToUse}
	keyWordList := make(map[string]string)
	keyChannel := make(chan Keyword)

	fmt.Printf("Amazon KeyWord Collector Started. Collect %d relevant keywords for the keyword '%s' \n", *limit, *keywordToUse)

	go requestKeyWords(keyChannel, keyword)

	for item := range keyChannel {
		if len(keyWordList) >= *limit {
			break
		}
		if _, ok := keyWordList[item.Keyword]; !ok {
			keyWordList[item.Keyword] = ""
			wg.Add(1)
			go func(item Keyword) {
				defer wg.Done()
				concurrentGoroutines <- struct{}{}
				requestKeyWords(keyChannel, item)
				<-concurrentGoroutines
			}(item)
		}
	}

	records := [][]string{
		{"#", "key_words"},
	}
	csvFile, err := os.Create(*keywordToUse + ".csv")
	if err != nil {
		log.Fatalf("Failed creating file: %s", err)
	}
	csvwriter := csv.NewWriter(csvFile)
	count := 1
	for key := range keyWordList {
		records = append(records, []string{strconv.Itoa(count), key})
		count++
	}
	csvwriter.WriteAll(records)

	fmt.Printf("Result: %d keywords related to the keyword '%s' were saved to the '%s.csv' file \n", len(keyWordList), *keywordToUse, *keywordToUse)
}

func requestKeyWords(keyChannel chan Keyword, keyword Keyword) {
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
	q.Add("prefix", keyword.Keyword)
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

	for _, item := range result.Suggestions {
		keyChannel <- Keyword{item.Value}
	}
}
