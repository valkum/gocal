package main

import (
    "fmt"
    "strings"
    // "bytes"
    //"encoding/xml"
    "net/http"
    // "net/http/cookiejar"
    "net/url"
    "flag"
    "strconv"
    "io/ioutil"
    "time"
    spew "github.com/davecgh/go-spew/spew"
)

var baseUrl = "https://www.campus.rwth-aachen.de/"
var baseUrlHttp = "http://www.campus.rwth-aachen.de/"
var homePath = "office/default.asp"
var loginPath = "office/views/campus/redirect.asp"
var calPath = "office/views/calendar/iCalExport.asp"
var logoutPath = "office/system/login/logoff.asp"
var roomPath = "rwth/all/room.asp"

type myCookieJar struct {
    cookies []*http.Cookie
}

func (c *myCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
    // spew.Dump(u, cookies)
    if c.cookies == nil {
        c.cookies = make([]*http.Cookie, 0)
    }

    for _, it := range cookies {
        c.cookies = append(c.cookies, it)
    }
}

func (c *myCookieJar) Cookies(u *url.URL) []*http.Cookie {
    // spew.Dump(c)
    return c.cookies
}

func testLog (req *http.Request, via []*http.Request) error {
    // spew.Dump(req)
    // spew.Dump(via)
    return nil
}
var cookieJar *myCookieJar
func main() {
    //cookieJar, _ := cookiejar.New(nil)
    cookieJar = &myCookieJar{}
    client := &http.Client{
        // Jar: cookieJar,
        // CheckRedirect : testLog,
    }

    username := flag.String("username", "", "MTR #")
    password := flag.String("password", "", "Password")
    flag.Usage=func() {
      fmt.Printf("Syntax:\n\tgocal [flags]\nwhere flags are:\n")
      flag.PrintDefaults()
    }
    flag.Parse()
    //if flag.NFlag() != 2 {
    //    flag.Usage()
    //    return
    //}
    //fmt.Printf("Loggin in with %s:%s\n", *username, *password)
    // var resp string
    // var head http.Header
    request("get", baseUrl + homePath, client, nil)
    // fmt.Printf("\n", head)
    v := url.Values{}
    v.Set("login", "> Login")
    v.Set("p", *password)
    v.Set("u", *username)
    v.Set("regwaygguid","")
    v.Set("evgguid","")

    // request("post", baseUrl + loginPath, client, v)
    resp, head, _ := request("post", baseUrl + loginPath, client, v)
    spew.Dump(head)
    fmt.Printf("%s\n", resp)
    startDate := time.Now()
    startDate = startDate.AddDate(0, 0, -7)
    endDate := time.Now()
    endDate = endDate.AddDate(0, 6, 0)
    v = url.Values{}
    v.Set("stadtdt", startDate.Format("02.01.2006"))
    v.Set("enddt", endDate.Format("02.01.2006"))

    // request("get", baseUrl + calPath, client, v)
    //fmt.Printf("\n", resp)

}


func request(method string, url string, cl *http.Client, params url.Values) (string, http.Header, http.Request) {

    var err error
    var req *http.Request
    // ...
    //req.Header.Add("If-None-Match", `W/"wyzzy"`)

    if(method == "post") {
        req, err = http.NewRequest("POST", url, strings.NewReader(params.Encode()))
        req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    } else if(method == "get") {
        req, err = http.NewRequest("GET", url + "?" + params.Encode(), nil)
    }
    req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/34.0.1847.116 Safari/537.36")
    req.Header.Add("Host", "www.campus.rwth-aachen.de")
    req.Header.Add("Origin", baseUrl + )
    for _, c := range cookieJar.cookies {
        req.AddCookie(c)
    }
    spew.Dump(req)

    var resp *http.Response
    resp, err = cl.Do(req)
    // if cl.Jar != nil {
        if rc := readSetCookies(resp.Header); len(rc) > 0 {
            cookieJar.SetCookies(req.URL, rc)
        }
    // }
    // spew.Dump(resp.Header)
    defer resp.Body.Close()

    if err != nil {
        fmt.Printf("Error from client.Do: %s\n", err)
    }
    var bodyString string
    //var err2 error
    if resp.StatusCode == 200 { // OK
        bodyBytes, err2 := ioutil.ReadAll(resp.Body)
        bodyString = string(bodyBytes)
        if err2 != nil {
            fmt.Printf("Error from ioutil.ReadAll: %s\n", err2)
        }
    }

    return bodyString, resp.Header, *req

}


// func fix_cookies(jar *cookiejar.Jar) {
//     u, _ := url.Parse(baseUrl + "office")
//     fmt.Print(u)
//     cookies := jar.Cookies(u)
//     submap := jar.entries
//     spew.Dump(submap)
//     for _,element := range cookies {
//         spew.Dump(*element)
//       element.HttpOnly = false
//       //send := element.shouldSend(true, host, path string
//     }

// }



// readSetCookies parses all "Set-Cookie" values from
// the header h and returns the successfully parsed Cookies.
func readSetCookies(h http.Header) []*http.Cookie {
    cookies := []*http.Cookie{}
    for _, line := range h["Set-Cookie"] {
        // fmt.Println("test")
        parts := strings.Split(strings.TrimSpace(line), ";")
        if len(parts) == 1 && parts[0] == "" {
            continue
        }
        // fmt.Println("Empty check passed")
        parts[0] = strings.TrimSpace(parts[0])
        j := strings.Index(parts[0], "=")
        if j < 0 {
            continue
        }
        name, value := parts[0][:j], parts[0][j+1:]
        if !isCookieNameValid(name) {
            continue
        }
        // fmt.Println("isValidName")
        value, success := parseCookieValue(value)
        if !success {
            continue
        }
        // fmt.Printf("value parsed for %s\n", name)
        c := &http.Cookie{
            Name:  name,
            Value: value,
            Raw:   line,
        }
        for i := 1; i < len(parts); i++ {
            parts[i] = strings.TrimSpace(parts[i])
            if len(parts[i]) == 0 {
                continue
            }

            attr, val := parts[i], ""
            if j := strings.Index(attr, "="); j >= 0 {
                attr, val = attr[:j], attr[j+1:]
            }
            lowerAttr := strings.ToLower(attr)
            parseCookieValueFn := parseCookieValue
            if lowerAttr == "expires" {
                parseCookieValueFn = parseCookieExpiresValue
            }
            val, success = parseCookieValueFn(val)
            if !success {
                c.Unparsed = append(c.Unparsed, parts[i])
                continue
            }
            switch lowerAttr {
            case "secure":
                c.Secure = true
                continue
            case "httponly":
                c.HttpOnly = true
                continue
            case "domain":
                c.Domain = val
                // TODO: Add domain parsing
                continue
            case "max-age":
                secs, err := strconv.Atoi(val)
                if err != nil || secs != 0 && val[0] == '0' {
                    break
                }
                if secs <= 0 {
                    c.MaxAge = -1
                } else {
                    c.MaxAge = secs
                }
                continue
            case "expires":
                c.RawExpires = val
                exptime, err := time.Parse(time.RFC1123, val)
                if err != nil {
                    exptime, err = time.Parse("Mon, 02-Jan-2006 15:04:05 MST", val)
                    if err != nil {
                        c.Expires = time.Time{}
                        break
                    }
                }
                c.Expires = exptime.UTC()
                continue
            case "path":
                c.Path = val
                // TODO: Add path parsing
                continue
            }
            c.Unparsed = append(c.Unparsed, parts[i])
        }
        cookies = append(cookies, c)
    }
    return cookies
}


func isCookieExpiresByte(c byte) (ok bool) {
    return isCookieByte(c) || c == ',' || c == ' '
}

func parseCookieValue(raw string) (string, bool) {
    return parseCookieValueUsing(raw, isCookieByte)
}

func parseCookieExpiresValue(raw string) (string, bool) {
    return parseCookieValueUsing(raw, isCookieExpiresByte)
}

func parseCookieValueUsing(raw string, validByte func(byte) bool) (string, bool) {
    raw = unquoteCookieValue(raw)
    for i := 0; i < len(raw); i++ {
        if !validByte(raw[i]) {
            return "", false
        }
    }
    return raw, true
}

func isCookieNameValid(raw string) bool {
    return true
}
func isCookieByte(c byte) bool {
    switch {
    case c == 0x21, 0x23 <= c && c <= 0x2b, 0x2d <= c && c <= 0x3a,
        0x3c <= c && c <= 0x5b, 0x5d <= c && c <= 0x7e:
        return true
    }
    return false
}

func unquoteCookieValue(v string) string {
    if len(v) > 1 && v[0] == '"' && v[len(v)-1] == '"' {
        return v[1 : len(v)-1]
    }
    return v
}
