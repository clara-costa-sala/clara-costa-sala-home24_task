package main

import (
    "bytes"
    "golang.org/x/net/html"
    "html/template"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "strings"
)

type PageInfo struct {
    WebsiteUrl string
    HtmlVersion string
    PageTitle string
    HeadingsCount map[string]int
    InternalLinks int
    ExternalLinks int
    BrokenLinks int
    Login string
}

func main() {
    http.HandleFunc("/", handler)
    http.HandleFunc("/checkUrl", checkUrl)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {

    if r.Method == http.MethodGet {
        tmpl := template.Must(template.ParseFiles("view.html"))
        err := tmpl.Execute(w, nil)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
        return
    }
}

func checkUrl(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        tmpl := template.Must(template.ParseFiles("view.html"))
        err := r.ParseForm()
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        siteUrl := r.FormValue("url")
        if validateUrl(siteUrl) {
            if webIsReachable(siteUrl) {
                //tmpl := template.Must(template.New("results").ParseFiles("view.html"))
                //tmpl, err = template.New("results").ParseFiles("view.html")
                pageInfo := getInfo(siteUrl)
                if err := tmpl.ExecuteTemplate(w, "results", pageInfo); err != nil {
                    http.Error(w, err.Error(), http.StatusInternalServerError)
                }
            } else {
                tmpl.Execute(w, struct{ NotReachable bool }{true})
            }
        } else {
            tmpl.Execute(w, struct{ WrongUrl bool }{true})
        }
    }
    http.Error(w, "", http.StatusBadRequest)
    /*
    if r.Method != http.MethodPost {
        tmpl.Execute(w, nil)
        return
    }

    tmpl.Execute(w, struct{ Success bool }{true})*/
}

func webIsReachable(siteUrl string) bool {
    r, e := http.Head(siteUrl)
    return e == nil && r.StatusCode == 200
}

func validateUrl(siteUrl string) bool{
    u, err := url.ParseRequestURI(siteUrl)
    return ! (err != nil || u.Scheme == "" || u.Host == "")
}

/*
   - TML Version
   - Page Title
   - Headings count by level
   - Amount of internal and external links
   - Amount of inaccessible links
   - If a page contains a login form
*/
func getInfo(siteUrl string) PageInfo {

    var pageInfo PageInfo
    version, title, headingsCount, links, loginForm := GetTagsInfo(siteUrl)
    externalLinks, internalLinks, brokenLinks := CheckLinks(siteUrl, links)

    pageInfo = PageInfo{
        siteUrl,
        version,
        title,
        headingsCount,
        internalLinks,
        externalLinks,
        brokenLinks,
        loginForm,
    }

    return pageInfo
}

func GetTagsInfo(siteUrl string) (string, string, map[string]int, []string, string) {
    title := "could not find tile"

    var links []string
    uniquelinks := make(map[string]bool)

    headingsCount := make(map[string]int)
    headings := []string{ "h1", "h2", "h3", "h4", "h5", "h6", "h7", "h8"}
    for _, heading := range headings { headingsCount[heading] = 0 }

    loginForm := "no login form"

    // get website content
    resp, err := http.Get(siteUrl)
    if err != nil {
        log.Println(err)
    }
    // close
    defer resp.Body.Close()

    content, err := ioutil.ReadAll(resp.Body)
    html_content := string(content)
    if err != nil {
        log.Println(err)
    }
    //reset the response body to the original unread state
    resp.Body = ioutil.NopCloser(bytes.NewBuffer(content))


    foundTitle := false
    foundLink := false
    foundForm := false
    foundButton := true
    tz := html.NewTokenizer(resp.Body)

    for {
        tt := tz.Next()
        token := tz.Token()
        err := tz.Err()
        if err == io.EOF {
            break
        }
        switch tt {
        case html.ErrorToken:
            log.Fatal(err)
        case html.StartTagToken, html.SelfClosingTagToken:
            // If it title, check for Text token to get title content
            if token.Data == "title" {
                foundTitle = true
            }
            // If it a data href="LINK"
            if token.Data == "a" {
                for _, attr := range token.Attr {
                    if attr.Key == "href" {
                        // Remove spaces in case
                        val := strings.ReplaceAll(attr.Val, " ", "")
                        if val == "#" {
                            foundLink = true
                        } else if uniquelinks[val] != true || foundLink {
                            if foundLink { foundLink = false }
                            links = append(links, val)
                            // Make sure we do not save the same URL > 1
                            uniquelinks[val] = true
                        }
                    }
                }
            }
            // If there is an input tage inside form, probably is login form
            if token.Data == "input" && foundForm {
                for _, attr := range token.Attr {
                    if attr.Key == "type" {
                        // if there is a password we consider is a login form
                        if attr.Val == "password" {
                            loginForm = "Exists"
                        }
                    }
                }
            }
            // if there is button we consider is a login form if the button says "login"
            if token.Data == "button" && foundForm {
                for _, attr := range token.Attr {
                    if attr.Key == "name" {
                        if attr.Val == "login" {
                            loginForm = "Exists"
                        }
                    }
                }
                foundButton = true
            }
            if token.Data == "form" {
                foundForm = true
            }
            for _, h := range headings {
                if strings.ToLower(token.Data) == h {
                    headingsCount[h] += 1
                }
            }
        case html.TextToken:
            if foundTitle {
                title = token.Data
                //fmt.Println("title -" + title)
                foundTitle = false
            }
            if foundButton && strings.Contains(strings.ToLower(token.Data), "log in" ) {
                loginForm = "Exists"
            }
        }
    }

    // Get first <>
    html_content_ := strings.Split(html_content, "<html")[0]
    // compare with doctypes and get version
    version := CheckHtmlVersion(strings.ToUpper(html_content_))

    return version, title, headingsCount, links, loginForm
}

func CheckLinks(siteUrl string, links []string) (int, int, int) {
    externalLinks, internalLinks, brokenLinks := 0, 0, 0

    base, err := url.Parse(siteUrl)
    if err != nil {
        log.Println(err)
    }
    //fmt.Println(siteUrl)
    for _, link := range links {
        if validateUrl(link) && ! strings.Contains(link, siteUrl) {
            //fmt.Println(" - External Link - " + link)
            externalLinks += 1
        } else {
            //fmt.Println(" - Internal Link - " + link)
            internalLinks += 1
            linkUrl, err := base.Parse(link)
            if err != nil {
                log.Panic(err)
            }
            link = linkUrl.String()
        }
        if IsBrokenLink(link) {
            // fmt.Println(" - Broken Link - " + link)
            brokenLinks += 1
        }
    }
    return externalLinks, internalLinks, brokenLinks
}

func IsBrokenLink(link string) bool {
    response, errors := http.Head(link)
    return ! (errors == nil && response.StatusCode == 200)
}

func CheckHtmlVersion(html_content string) string {
    doctypes := map[string]string {
        "HTML 4.01 Strict": "HTML 4.01",
        "HTML 4.01 Transitional": "HTML 4.01 Transitional",
        "HTML 4.01 Frameset": "HTML 4.01 Frameset",
        "XHTML 1.0 Strict": "XHTML 1.0 Strict",
        "XHTML 1.0 Transitional": "XHTML 1.0 Transitional",
        "XHTML 1.0 Frameset": "XHTML 1.0 Frameset",
        "XHTML 1.1": "XHTML 1.1",
        "HTML 5": "<!DOCTYPE HTML>",
    }

    version := "[blank]"
    for doctype, matcher := range doctypes {
        if strings.Contains(html_content, matcher) {
            version = doctype
        }
    }
    return version
}
