package main

import (
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/gosuri/uiprogress"
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
	Keyword          string
	TotalResultCount int64
}

func DoWork() {
	time.Sleep(500 * time.Millisecond)
}

var (
	g = color.New(color.FgHiGreen)
	y = color.New(color.FgHiYellow)
	r = color.New(color.FgHiRed)
)

func main() {
	// Init progress bar
	uiprogress.Start()

	// Flags
	concurency := flag.Int("concurency", 5, "the number of goroutines that are allowed to run concurrently")
	limit := flag.Int("limit", 100, "number of keywords to collect")
	keywordToUse := flag.String("keyword", "", "keyword to use")
	flag.Parse()

	if *keywordToUse == "" {
		r.Println("KeyWord is missing. To view help enter: akrt -help")
		os.Exit(1)
	}
	g.Printf("Collect %d relevant keywords to the keyword '%s' \n", *limit, *keywordToUse)

	// Keyword Collector progress bar
	keywordBar := uiprogress.AddBar(*limit).AppendCompleted().PrependElapsed()
	keywordBar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Keywords (%d/%d)", b.Current(), *limit)
	})

	// Limiting concurent requests to collect keywords
	concurrentGoroutines := make(chan struct{}, *concurency)

	keyword := Keyword{*keywordToUse, 0}
	keyWordList := make(map[string]Keyword)
	keyChannel := make(chan Keyword)

	go requestKeyWords(keyChannel, keyword)

	toLongKeys := 0
	for item := range keyChannel {
		if len(keyWordList) >= *limit {
			break
		}
		if toLongKeys > 10 {
			break
		}
		if item.Keyword == "" {
			toLongKeys++
		} else {
			if _, ok := keyWordList[item.Keyword]; !ok {
				keywordBar.Incr()
				keyWordList[item.Keyword] = item
				go func(item Keyword) {
					concurrentGoroutines <- struct{}{}
					requestKeyWords(keyChannel, item)
					<-concurrentGoroutines
				}(item)
			}
		}
	}
	// Limiting concurent requests to collect number of products per keyword
	concurrentGoroutinesProductCount := make(chan struct{}, 5)
	totalResultCount := make(chan Keyword)

	// Product Count Collector progress bar
	productCountBar := uiprogress.AddBar(len(keyWordList)).AppendCompleted().PrependElapsed()
	productCountBar.PrependFunc(func(b *uiprogress.Bar) string {
		return fmt.Sprintf("Product Count (%d/%d)", b.Current(), len(keyWordList))
	})

	for key := range keyWordList {
		go func(item Keyword) {
			concurrentGoroutinesProductCount <- struct{}{}
			keywordMetadata(totalResultCount, item)
			<-concurrentGoroutinesProductCount
		}(keyWordList[key])
	}
	products := 0
	for item := range totalResultCount {
		products++
		productCountBar.Incr()
		keyWordList[item.Keyword] = item
		if products >= len(keyWordList) {
			close(totalResultCount)
		}
	}

	records := [][]string{
		{"#", "key_words", "total_products"},
	}
	csvFile, err := os.Create(*keywordToUse + ".csv")
	if err != nil {
		log.Fatalf("Failed creating file: %s", err)
	}
	csvwriter := csv.NewWriter(csvFile)
	count := 1
	for key := range keyWordList {
		totalProducts := strconv.FormatInt(keyWordList[key].TotalResultCount, 10)
		records = append(records, []string{strconv.Itoa(count), keyWordList[key].Keyword, totalProducts})
		count++
	}
	csvwriter.WriteAll(records)

	y.Printf("Collected %d keywords: '%s.csv' \n", len(keyWordList), *keywordToUse)

}

func requestKeyWords(keyChannel chan Keyword, keyword Keyword) {
	client := http.Client{}

	req, _ := http.NewRequest("GET", "https://completion.amazon.com/api/2017/suggestions", nil)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_"+strconv.Itoa(rand.Intn(15-9)+9)+"_1) AppleWebKit/531.36 (KHTML, like Gecko) Chrome/"+strconv.Itoa(rand.Intn(79-70)+70)+".0.3945.130 Safari/531.36")
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

	if len(result.Suggestions) == 1 {
		keyChannel <- Keyword{"", 0}
	} else {
		for _, item := range result.Suggestions {
			keyChannel <- Keyword{item.Value, 0}
		}
	}
}

func keywordMetadata(totalResultCount chan Keyword, keyword Keyword) {
	client := http.Client{}

	req, _ := http.NewRequest("GET", "https://www.amazon.com/s", nil)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_"+strconv.Itoa(rand.Intn(15-9)+9)+"_1) AppleWebKit/531.36 (KHTML, like Gecko) Chrome/"+strconv.Itoa(rand.Intn(79-70)+70)+".0.3945.130 Safari/531.36")
	req.Header.Set("Origin", "https://www.amazon.com")
	req.Header.Set("Referer", "https://www.amazon.com/")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,ru;q=0.8")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")

	q := req.URL.Query()
	q.Add("i", "aps")
	q.Add("k", keyword.Keyword)
	q.Add("ref", "nb_sb_noss")
	q.Add("url", "search-alias=aps")
	req.URL.RawQuery = q.Encode()

	r, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
	}
	var reader io.ReadCloser
	reader, _ = gzip.NewReader(r.Body)

	dataInBytes, _ := ioutil.ReadAll(reader)
	pageContent := string(dataInBytes)

	reTotalCount := regexp.MustCompile(`(\w*"totalResultCount":\w*)(.[0-9])`)
	res := reTotalCount.FindAllString(string(pageContent), -1)

	var total int64 = 0
	if len(res) > 0 {
		reCount := regexp.MustCompile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
		submatchall := reCount.FindAllString(res[0], -1)
		total, _ = strconv.ParseInt(submatchall[0], 0, 64)
	}

	totalResultCount <- Keyword{keyword.Keyword, total}
}
