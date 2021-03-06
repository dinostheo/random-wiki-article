package randomwiki

import (
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

func Generate(language string) (string, []string) {
	visited = make(map[string]bool)
	nonHTML = make(map[string]bool)

	baseDomain := fmt.Sprintf("https://%s.wikipedia.org", language)
	initialPoint := baseDomain + "/wiki/Main_Page"

	visited[initialPoint] = true

	rand.Seed(time.Now().UTC().UnixNano())
	count = 0

	graphState := make([]string, 0)

	return crawl(baseDomain, initialPoint, graphState)
}
