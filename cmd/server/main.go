/*
It randomly navigates to a localized wikipedia page starting from the main page.
It will follow 10 random wikipedia links and it will return the url of a random article.
If no language is provided it will use the english language.

An example request for an english article is the following:

	http://localhost:8080/wiki/en
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dinostheo/random-wiki-article/pkg/randomwiki"
	"net/http"
	"os"
	"strings"
)

var languagePath = "languages.json"

type result struct {
	URL   string   `json:"url"`
	Graph []string `json:"graph"`
}

type languages []struct {
	English string `json:"English"`
	Alpha2  string `json:"alpha2"`
}

func loadLanguageList(path string) map[string]string {
	jsonFile, err := os.Open(path)

	if err != nil {
		fmt.Println("Failed to read languages.json ", err)
		os.Exit(1)
	}

	defer jsonFile.Close()

	var langs languages

	json.NewDecoder(jsonFile).Decode(&langs)

	languageList := make(map[string]string)

	for _, lang := range langs {
		languageList[lang.Alpha2] = lang.English
	}

	return languageList
}

func checkLanguageCode(code string, path string) bool {
	languageList := loadLanguageList(path)

	if _, ok := languageList[code]; ok {
		return true
	}

	return false
}

func generateRandomArticle() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		language := strings.SplitN(r.URL.Path, "/", 3)[2]

		if language == "" {
			language = "en"
		}

		if !checkLanguageCode(language, languagePath) {
			err := errors.New("Invalid language code: " + language)

			http.Error(w, err.Error(), http.StatusBadRequest)

			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		wikiURL, pathToArticle := randomwiki.Generate(language)

		data := result{
			wikiURL,
			pathToArticle,
		}

		json.NewEncoder(w).Encode(data)
	})
}

func main() {
	http.Handle("/wiki/", generateRandomArticle())

	http.ListenAndServe(":8080", nil)
}
