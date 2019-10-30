package main

import (
	"fmt"
	"net/http"
	"os"

	"gopkg.in/headzoo/surf.v1"
)

func require(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	fmt.Println("web")

	bow := surf.NewBrowser()

	require(bow.Open("https://portal.discover.com/customersvcs/universalLogin/ac_main"))

	before := bow.SiteCookies()
	fmt.Println("cookies before\n", before)

	form, err := bow.Form("#login-form-content")
	require(err)

	require(form.Input("userID", "johnmstarich"))
	require(form.Input("password", os.Getenv("PASS")))
	require(form.Submit())

	//values := make(url.Values)
	//values.Add("userID", "johnmstarich")
	//values.Add("password-content", os.Getenv("PASS"))

	//require(bow.PostForm("https://portal.discover.com/customersvcs/universalLogin/signin", values))

	after := bow.SiteCookies()
	fmt.Println("cookies after\n", after)

	beforeMap := make(map[string]*http.Cookie)
	for _, c := range before {
		beforeMap[c.Name] = c
	}
	for _, c := range after {
		if beforeMap[c.Name] == nil {
			fmt.Println("new cookies", c)
		}
	}

	fmt.Println("signin headers\n", bow.ResponseHeaders(), bow.Body())

	//fmt.Println(bow.Title(), bow.Body())
	require(bow.Open("https://card.discover.com/cardmembersvcs/achome/homepage"))
	require(bow.Open("https://card.discover.com/cardmembersvcs/statements/app/activity"))

	//bow.SetAttributes
	require(bow.Open("https://card.discover.com/cardmembersvcs/ofxdl/ofxWebDownload?stmtKey=CTD&startDate=20191007&endDate=20191007&fileType=QFX&bid=9625&fileName=Discover-RecentActivity-20191007.qfx"))

	fmt.Println("download result\n", bow.ResponseHeaders(), "\n", bow.Body())
}
