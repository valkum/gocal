package main

import (
    "fmt"
    "bytes"
    //"encoding/xml"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "flag"
    "io/ioutil"
    "time"
    //spew "github.com/davecgh/go-spew/spew"
)

var baseUrl = "https://www.campus.rwth-aachen.de/"
var homePath = "office/default.asp"
var loginPath = "office/views/campus/redirect.asp"
var calPath = "office/views/calendar/iCalExport.asp"
var logoutPath = "office/system/login/logoff.asp"
var roomPath = "rwth/all/room.asp"



func main() {
    username := flag.String("username", "123456", "MTR #")
    password := flag.String("password", "######", "Password")

    flag.Parse()


    cookieJar, _ := cookiejar.New(nil)
    request("get", baseUrl + homePath, cookieJar, nil)
    v := url.Values{}
    v.Set("login", "> Login")
    v.Set("p", *password)
    v.Set("u", *username)
    request("post", baseUrl + loginPath, cookieJar, v)

    startDate := time.Now()
    startDate = startDate.AddDate(0, 0, -7)
    endDate := time.Now()
    endDate = endDate.AddDate(0, 6, 0)
    v = url.Values{}
    v.Set("stadtdt", startDate.Format("02.01.2006"))
    v.Set("enddt", endDate.Format("02.01.2006"))
    resp, _ := request("get", baseUrl + calPath, cookieJar, v)
    fmt.Printf("%s\n", resp)


}


func request(method string, url string, jar *cookiejar.Jar, params url.Values) (string, http.Request) {
    client := &http.Client{
        Jar: jar,
    }

    var err error
    var req *http.Request
    req, err = http.NewRequest("GET", url, bytes.NewBufferString(params.Encode()))
    // ...
    //req.Header.Add("If-None-Match", `W/"wyzzy"`)

    if(method == "post") {
        req, err = http.NewRequest("POST", url, bytes.NewBufferString(params.Encode()))
    }
    var resp *http.Response
    resp, err = client.Do(req)
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


    return bodyString, *req

}
