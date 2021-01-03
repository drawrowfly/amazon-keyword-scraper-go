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
	"sync"

	"github.com/fatih/color"
	"github.com/gosuri/uiprogress"
)

var (
	g     = color.New(color.FgHiGreen)
	y     = color.New(color.FgHiYellow)
	r     = color.New(color.FgHiRed)
	muCtx sync.RWMutex
)

var Mids = map[string]string{
	"com":    "ATVPDKIKX0DER",
	"ca":     "A2EUQ1WTGCTBG2",
	"co.uk":  "A1F83G8C2ARO7P",
	"com.au": "A39IBJ37TRP1C6",
	"de":     "A1PA6795UKMFR9",
}

func main() {
	// Init progress bar
	uiprogress.Start()

	// Flags
	concurency := flag.Int("concurency", 5, "the number of goroutines that are allowed to run concurrently")
	limit := flag.Int("limit", 100, "number of keywords to collect")
	keywordToUse := flag.String("keyword", "", "keyword to use")
	domainToSearch := flag.String("tld", "com", "Amazon TLD domain to search; com, ca, co.uk, de, com.au")
	flag.Parse()

	if *keywordToUse == "" {
		r.Println("KeyWord is missing. To view help enter: akrt -help")
		os.Exit(1)
	}
	g.Printf("[amazon.%s] Collect %d relevant keywords to the keyword '%s' \n", *domainToSearch, *limit, *keywordToUse)

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
	// Initially there's just 1 keyword
	context := Context{1, *domainToSearch}

	go requestKeyWords(keyChannel, keyword, &context)

	for item := range keyChannel {
		if len(keyWordList) >= *limit {
			break
		}

		if item.Keyword != "" {
			if _, ok := keyWordList[item.Keyword]; !ok {
				keywordBar.Incr()
				keyWordList[item.Keyword] = item
				go func(item Keyword) {
					concurrentGoroutines <- struct{}{}
					requestKeyWords(keyChannel, item, &context)
					<-concurrentGoroutines
				}(item)
			} else {
				// decrement total count, because this suggestion is already in the list
				muCtx.Lock()
				context.KeywordsFound--
				muCtx.Unlock()
			}
		}

		muCtx.RLock()
		keywordsFound := context.KeywordsFound
		muCtx.RUnlock()
		if keywordsFound <= 0 {
			break
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
			keywordMetadata(totalResultCount, item, &context)
			<-concurrentGoroutinesProductCount
		}(keyWordList[key])
	}
	products := 0
	productCountBar.Incr()
	for item := range totalResultCount {
		productCountBar.Incr()
		products++
		keyWordList[item.Keyword] = item
		if products >= len(keyWordList) {
			close(totalResultCount)
		}
	}

	// Saving result to the CSV file
	records := [][]string{
		{"#", "key_words", "total_products"},
	}
	filename := fmt.Sprintf("%s_%s.csv", *keywordToUse, context.TLD)
	csvFile, err := os.Create(filename)
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

	y.Printf("Collected %d keywords: '%s' \n", len(keyWordList), filename)

}

func requestKeyWords(keyChannel chan Keyword, keyword Keyword, context *Context) {
	client := http.Client{}

	req, _ := http.NewRequest("GET", fmt.Sprintf("https://completion.amazon.%s/api/2017/suggestions", context.TLD), nil)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_"+strconv.Itoa(rand.Intn(15-9)+9)+"_1) AppleWebKit/531.36 (KHTML, like Gecko) Chrome/"+strconv.Itoa(rand.Intn(79-70)+70)+".0.3945.130 Safari/531.36")

	q := req.URL.Query()
	q.Add("mid", Mids[context.TLD])
	q.Add("alias", "aps")
	q.Add("fresh", "0")
	q.Add("ks", "88")
	q.Add("prefix", keyword.Keyword)
	q.Add("event", "onKeyPress")
	q.Add("limit", "11")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
	}

	var result KeywordSuggestions

	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Suggestions) == 0 {
		log.Fatalln("No keywords found")
	}

	// Substract look up keyword and add suggestions, if any
	muCtx.Lock()
	if len(result.Suggestions) == 1 {
		context.KeywordsFound--
	} else {
		context.KeywordsFound += len(result.Suggestions) - 1
	}
	muCtx.Unlock()

	if len(result.Suggestions) == 1 {
		keyChannel <- Keyword{"", 0}
	} else {
		for _, item := range result.Suggestions {
			keyChannel <- Keyword{item.Value, 0}
		}
	}
}

func keywordMetadata(totalResultCount chan Keyword, keyword Keyword, context *Context) {
	client := http.Client{}

	req, _ := http.NewRequest("GET", fmt.Sprintf("https://www.amazon.%s/s", context.TLD), nil)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_"+strconv.Itoa(rand.Intn(15-9)+9)+"_1) AppleWebKit/531.36 (KHTML, like Gecko) Chrome/"+strconv.Itoa(rand.Intn(79-70)+70)+".0.3945.130 Safari/531.36")
	req.Header.Set("Origin", fmt.Sprintf("https://www.amazon.%s", context.TLD))
	req.Header.Set("Referer", fmt.Sprintf("https://www.amazon.%s", context.TLD))
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
