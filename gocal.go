package main

import (
    "fmt"
    "strings"
    "bufio"
    // "strconv"
    "net/http"
    // "net/http/cookiejar"
    "net/url"
    "flag"
    "io/ioutil"
    "regexp"
    "time"
    "unicode/utf8"
    "log"
    "os"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    mahonia "github.com/axgle/mahonia"
    spew "github.com/davecgh/go-spew/spew"
)

var baseUrl = "https://www.campus.rwth-aachen.de/"
var baseUrlHttp = "http://www.campus.rwth-aachen.de/"
var homePath = "office/default.asp"
var loginPath = "office/views/campus/redirect.asp"
var calPath = "office/views/calendar/iCalExport.asp"
var logoutPath = "office/system/login/logoff.asp"
var roomPath = "rwth/all/room.asp"


type Room struct {
    id          string
    address     string
    cluster     string
    building    string
    building_no string
    room        string
    room_no     string
    floor       string
}


var cookieJar *myCookieJar
var client *http.Client
func main() {
    //cookieJar, _ := cookiejar.New(nil)
    cookieJar = &myCookieJar{}
    client = &http.Client{
        Jar: cookieJar,
        // CheckRedirect : testLog,
    }

    username := flag.String("username", "", "MTR #")
    password := flag.String("password", "", "Password")
    flag.Usage=func() {
      fmt.Printf("Syntax:\n\tgocal [flags]\nwhere flags are:\n")
      flag.PrintDefaults()
    }
    flag.Parse()
    if flag.NFlag() != 2 {
       flag.Usage()
       return
    }
    //fmt.Printf("Loggin in with %s:%s\n", *username, *password)
    var resp string
    // var head http.Header
    request("get", baseUrl + homePath, client, nil)
    // spew.Dump("\n", head)
    v := url.Values{}
    v.Set("login", "> Login")
    v.Set("p", *password)
    v.Set("u", *username)

    request("post", baseUrl + loginPath, client, v)
    // resp, head, _ = request("post", baseUrl + loginPath, client, v)
    // spew.Dump(head)
    // fmt.Printf("%s\n", resp)
    startDate := time.Now()
    startDate = startDate.AddDate(0, 0, -7)
    endDate := time.Now()
    endDate = endDate.AddDate(0, 6, 0)
    v = url.Values{}
    v.Set("startdt", startDate.Format("02.01.2006"))
    v.Set("enddt", endDate.Format("02.01.2006"))

    resp, _, _ = request("get", baseUrl + calPath, client, v)
    // fmt.Printf("%s\n", resp)
    // fmt.Println(resp)

    db, err := sql.Open("sqlite3", "./foo.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    _, err = db.Exec("CREATE TABLE IF NOT EXISTS rooms (id VARCHAR(255) PRIMARY KEY, address VARCHAR(255), cluster VARCHAR(255), building VARCHAR(255), building_no INTEGER, room VARCHAR(255), room_no INTEGER, floor VARCHAR(255))")
    if err != nil {
        log.Fatal(err)
    }

    // open output file
    fo, err := os.Create("C:/Users/valkum/Dropbox/public/318099.ics")
    if err != nil {
        log.Fatal(err)
    }
    // close fo on exit and check for its returned error
    defer func() {
        if err := fo.Close(); err != nil {
            panic(err)
        }
    }()
    w := bufio.NewWriter(fo)

    var address Room
    var category string
    lines := strings.Split(resp, "\r\n")
    for _,line := range lines {
        if len(strings.TrimSpace(line)) != 0 {
            if j := strings.Index(line, ":"); j >= 0 {
                key, value := line[:j], line[j+1:]
                switch key {
                case "END":
                    if value == "VEVENT" {
                        address = Room{}
                        category = ""
                    }
                    break
                case "CATEGORIES":
                    category = value
                    break
                case "LOCATION":
                    var reg = regexp.MustCompile(`^([0-9]+\|[0-9]+)`)
                    if matches := reg.FindAllString(value, -1); len(matches) != 0 {
                        room := matches[0]
                        address = get_address(db, room)

                        if address == (Room{}) {
                            fmt.Println("no address found")
                            address = crawl_address(room)
                            set_address(db, room, address)
                        }

                        if address.address != "" {
                            value = address.address + ", Aachen"
                        }

                    }
                    break
                case "DESCRIPTION":
                    additional := value
                    value = ""
                    if address.building != "" || address.building_no != "" {
                        value += "\nGebäude: " +  address.building_no + " " + address.building
                    }
                    if address.room != "" || address.room_no != "" {
                        value += "\nRaum: " +  address.room_no + " " + address.room
                    }
                    if address.floor != "" {
                        value += "\nGeschoss: " +  address.floor
                    }
                    if address.cluster != "" {
                        value += "\nCampus: " +  address.floor
                    }
                    if category != "" {
                        value += "\nTyp: " + category
                    }
                    if additional != "" && additional != "Kommentar" {
                        value += "\n"+additional
                    }
                    break
                }
                fmt.Fprint(w, key + ":" + strings.TrimSpace(value))
            }
        }
        fmt.Fprint(w, "\r\n")
    }
    w.Flush()
}


func request(method string, url string, cl *http.Client, params url.Values) (string, http.Header, http.Request) {

    var err error
    var req *http.Request
    // ...
    //req.Header.Add("If-None-Match", `W/"wyzzy"`)

    if(method == "post") {
        req, err = http.NewRequest("POST", url, strings.NewReader(params.Encode()))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    } else {
        req, err = http.NewRequest("GET", url + "?" + params.Encode(), nil)
    }
    req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/34.0.1847.116 Safari/537.36")
    req.Header.Set("Origin", "www.campus.rwth-aachen.de")
    req.Header.Set("Referer", "https://www.campus.rwth-aachen.de/office/views/campus/redirect.asp")
    req.Header.Set("Accept-Language", "de-DE,de;q=0.8,en-US;q=0.6,en;q=0.4")
    req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
    // for _, c := range cookieJar.cookies {
    //     req.AddCookie(c)
    // }

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
        bodyBytes, _ := ioutil.ReadAll(resp.Body)
        if utf8.Valid(bodyBytes) {
            bodyString = string(bodyBytes)
        } else {
            enc := mahonia.NewDecoder("latin-1")
            bodyString = enc.ConvertString(string(bodyBytes))
        }
    }

    return bodyString, resp.Header, *req

}

func get_address (db *sql.DB, r string) (result Room) {
    stmt, err := db.Prepare("SELECT * FROM rooms WHERE id = ?")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()
    result = Room{}
    err = stmt.QueryRow(r).Scan(&result.id, &result.address, &result.cluster, &result.building, &result.building_no, &result.room, &result.room_no, &result.floor)
    if err != nil {
        log.Fatalf("Error running %q: %v", stmt, err)
        spew.Dump(stmt)
    }
    return result
}
func set_address (db *sql.DB, room string, address Room) {
    stmt, err := db.Prepare("INSERT OR REPLACE INTO rooms (id, address, cluster, building, building_no, room, room_no, floor) values (?, ?, ?, ?, ?, ?, ?, ?)")
    if err != nil {
        log.Fatal(err)
    }
    defer stmt.Close()
    _, err = stmt.Exec(room, address.address, address.cluster, address.building, address.building_no, address.room, address.room_no, address.floor)
    if err != nil {
        log.Fatal(err)
    }
    return
}

func crawl_address (room string) (matches Room) {
    resp, _, _ := request("GET", baseUrl + roomPath, client, url.Values{"room": {room}})

    infos := map[string]string{
    "cluster" : "H.rsaalgruppe",
    "address" : "Geb.udeanschrift",
    "building" : "Geb.udebezeichnung",
    "building_no" : "Geb.udenummer",
    "room" : "Raumname",
    "room_no" : "Raumnummer",
    "floor" : "Geschoss",
    }

    for index, pattern := range infos {
        re := regexp.MustCompile("<td class=\"default\">" + pattern + "</td><td class=\"default\">([^<]*)</td>")
        match := re.FindStringSubmatch(resp);
        if ( len(match) != 0) {
            if matches == (Room{}) {
                matches = Room{}
            }
            re := regexp.MustCompile("/[ ]{2,}/sm")
            switch index {
            case "cluster":
                matches.cluster = re.ReplaceAllString(match[1], " ")
                break
            case "address":
                matches.address = re.ReplaceAllString(match[1], " ")
                break
            case "building":
                matches.building = re.ReplaceAllString(match[1], " ")
                break
            case "building_no":
                matches.building_no = re.ReplaceAllString(match[1], " ")
                break
            case "room":
                matches.room = re.ReplaceAllString(match[1], " ")
                break
            case "room_no":
                matches.room_no = re.ReplaceAllString(match[1], " ")
                break
            case "floor":
                matches.floor = re.ReplaceAllString(match[1], " ")
                break
            }
        }
    }

    return matches
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




