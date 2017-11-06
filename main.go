/*
It randomly navigates to a localized wikipedia page starting from the main page.
It will follow 10 random wikipedia links and it will return the url of a random article.
If no language is provided it will use the english language.

An example request for an english article is the following:

	http://localhost:8080/wiki/en
*/
package main

import (
	"bytes"
	"encoding/json"
	"errors"
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
var aHrefRegexp = regexp.MustCompile(`\/wiki\/[^` + strings.Join(whitelistStrings, "|") + `:]([\w\-\.,@?^=%&amp;\+#]*[\w\-\@?^=%&amp;\+#])`)

type result struct {
	URL   string   `json:"url"`
	Graph []string `json:"graph"`
}

type Languages []struct {
	English string `json:"English"`
	Alpha2  string `json:"alpha2"`
}

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

func getRandomURL(baseDomain string, urls []string) string {
	randomIndex := rand.Intn(len(urls))
	unEscapedPath, _ := url.PathUnescape(urls[randomIndex])
	wikiURL := baseDomain + unEscapedPath

	if visited[wikiURL] || nonHTML[wikiURL] {
		if len(urls) == 1 {
			p, _ := url.PathUnescape(urls[0])

			return baseDomain + p
		}

		urls = append(urls[:randomIndex], urls[randomIndex+1:]...)

		return getRandomURL(baseDomain, urls)
	}

	return wikiURL
}

func crawl(baseDomain string, urlStr string, graphState []string) (string, []string) {
	urls := findUrls(urlStr)

	if len(urls) == 0 {
		return urlStr, graphState
	}

	if count++; count >= 10 {
		return urlStr, graphState
	}

	randomURL := getRandomURL(baseDomain, urls)

	visited[randomURL] = true

	graphState = append(graphState, randomURL)

	return crawl(baseDomain, randomURL, graphState)
}

func generateRandomArticle(languageList map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited = make(map[string]bool)
		nonHTML = make(map[string]bool)

		language := strings.SplitN(r.URL.Path, "/", 3)[2]

		if language == "" {
			language = "en"
		}

		if _, ok := languageList[language]; !ok {
			err := errors.New("Invalid language code: " + language)

			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		baseDomain := fmt.Sprintf("https://%s.wikipedia.org", language)
		initialPoint := baseDomain + "/wiki/Main_Page"

		visited[initialPoint] = true

		rand.Seed(time.Now().UTC().UnixNano())
		count = 0

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		graphState := make([]string, 0)

		wikiURL, pathToArticle := crawl(baseDomain, initialPoint, graphState)

		data := result{
			wikiURL,
			pathToArticle,
		}

		json.NewEncoder(w).Encode(data)
	})
}

func main() {
	jsonData, err := ioutil.ReadFile("languages.json")

	if err != nil {
		fmt.Println("Failed to read languages.json")
		os.Exit(1)
	}

	var languages Languages

	json.NewDecoder(bytes.NewReader(jsonData)).Decode(&languages)

	languageList := make(map[string]string)

	for _, lang := range languages {
		languageList[lang.Alpha2] = lang.English
	}

	http.Handle("/wiki/", generateRandomArticle(languageList))

	http.ListenAndServe(":8080", nil)
}
