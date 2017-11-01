package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const baseDomain = "https://en.wikipedia.org"

var visited map[string]bool
var nonHTML map[string]bool
var count int
var whitelistStrings = []string{
	"Special",
	"Wikipedia",
	"Portal",
	"Template",
	"File",
	"User",
	"Help",
	"Category",
}

var contentTypeRegexp = regexp.MustCompile(`^text\/html`)
var aHrefRegexp = regexp.MustCompile(`\/wiki\/[^` + strings.Join(whitelistStrings, "|") + `:]([\w\-\.,@?^=%&amp;:/~\+#]*[\w\-\@?^=%&amp;/~\+#])`)

func findUrls(urlStr string) (result []string) {
	res, err := http.Get(urlStr)

	defer res.Body.Close()

	if err != nil {
		fmt.Println("Page fetch error: ", err)

		os.Exit(1)
	}

	if contentType := res.Header.Get("Content-Type"); !contentTypeRegexp.MatchString(contentType) {
		nonHTML[urlStr] = true

		return result
	}

	body, resError := ioutil.ReadAll(res.Body)

	if resError != nil {
		fmt.Println("Response reading error: ", resError)

		os.Exit(1)
	}

	pageStr := string(body)
	result = aHrefRegexp.FindAllString(pageStr, -1)

	return result
}

func getURLHostName(urlStr string) string {
	u, err := url.Parse(urlStr)

	if err != nil {
		fmt.Println("Url parser error: ", err)

		os.Exit(1)
	}

	return u.Hostname()
}

func getRandomURL(urls []string) string {
	randomIndex := rand.Intn(len(urls))

	wikiURL := baseDomain + urls[randomIndex]

	if visited[wikiURL] || nonHTML[wikiURL] {
		fmt.Println("--- Avoiding circulation or Non HTML --- ", wikiURL)

		if len(urls) == 1 {
			return baseDomain + urls[0]
		}

		urls = append(urls[:randomIndex], urls[randomIndex+1:]...)

		return getRandomURL(urls)
	}

	return wikiURL
}

func crawl(urlStr string) string {
	urls := findUrls(urlStr)

	if len(urls) == 0 {
		return urlStr
	}

	if count++; count >= 100 {
		return urlStr
	}

	randomURL := getRandomURL(urls)

	visited[randomURL] = true

	return crawl(randomURL)
}

func main() {
	visited = make(map[string]bool)
	nonHTML = make(map[string]bool)

	initialPoint := baseDomain + "/wiki/Main_Page"

	visited[initialPoint] = true

	http.HandleFunc("/wiki/", func(w http.ResponseWriter, r *http.Request) {
		rand.Seed(time.Now().UTC().UnixNano())
		count = 0

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		data := make(map[string]string)

		data["url"] = crawl(initialPoint)

		json.NewEncoder(w).Encode(data)
	})

	http.ListenAndServe(":8080", nil)
}
