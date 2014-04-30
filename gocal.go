package main

import (
  "fmt"
  "strings"
  "bufio"
  "database/sql"
  "net/http"
  "net/url"
  "io/ioutil"
  "regexp"
  "time"
  "unicode/utf8"
  "log"
  "os"
  gopass "github.com/howeyc/gopass"
  mahonia "github.com/axgle/mahonia")

var baseUrl = "https://www.campus.rwth-aachen.de/"
var baseUrlHttp = "http://www.campus.rwth-aachen.de/"
var homePath = "office/default.asp"
var loginPath = "office/views/campus/redirect.asp"
var calPath = "office/views/calendar/iCalExport.asp"
var logoutPath = "office/system/login/logoff.asp"
var roomPath = "rwth/all/room.asp"
var version = "0.1"


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

var db *Database
var e *Encryption

var cookieJar *myCookieJar
var client *http.Client

func main() {
  start()
  defer db.conn.Close()
  args :=  os.Args[1:len(os.Args)]
  for  _,arg := range args {
    if (arg == "-h" || arg == "--help") {
      fmt.Println("Usage: gocal [TASK]")
      fmt.Println("Lädt deinen CampusOffice Kalender in den gespeicherten Ordner und fügt Rauminformationen hinzu.")
      fmt.Println("Ähntlich zu dem Onlinetool CoCal")
      fmt.Println("Tasks: ")
      fmt.Printf("   setup \tRichtet goCal ein\n")
      fmt.Printf("   run \tLädt deinen Kalender runter\n")
      fmt.Println()
      fmt.Println()
      fmt.Printf("goCal version: %s", version)
      os.Exit(0)
    } else if (arg == "-v" || arg == "--version") {
      fmt.Printf("goCal version: %s", version)
      os.Exit(0)
    } else {
      switch (arg) {
        case "setup":
          setup()
          break
        case "run":
          calendar()
          break;
        default:
          fmt.Printf("Unbekannter Befehel: gocal %s\n", arg)
      }
    }
  }
  if(len(args) == 0) {
    calendar()
  }
}

func start() {
  cookieJar = &myCookieJar{}
  client = &http.Client{
      Jar: cookieJar,
  }
  db = database()
  db.init()
  e = encryption()
}

func setup() {
  var dir string
  var username string
  var password []byte

  fmt.Println("Setup")
  fmt.Println("--------------------")
  fmt.Println("Bitte gib deine CampusOffice daten ein.")
  fmt.Println("Deine Zugangsdaten werden verschlüsselt auf deinem Computer abgelegt.")

  fmt.Printf("MatrNr: ")
  fmt.Scanln(&username)
  fmt.Printf("Passwort: ")
  password = gopass.GetPasswdMasked()


  for !login(string(username), string(password)) {
    fmt.Println("Falche MatrNr oder falsches Passwort!")

    fmt.Printf("MatrNr: ")
    fmt.Scanln(&username)
    fmt.Printf("Passwort: ")
    password = gopass.GetPasswdMasked()
  }

  fmt.Println("Erfolgreich eingeloggt")
  db.set_encrypted_setting(e, "username", username)
  db.set_encrypted_setting(e, "password", string(password))

  fmt.Println("Bitte gib den Speicherort an, an dem deine iCal exportiert werden soll.")
  fmt.Println("Beispiele:")
  fmt.Println("Windows: C:\\Users\\Benutzer\\Dropbox")
  fmt.Println("Linux: /home/Benutzer/Dropbox")

  fmt.Printf("Speicherort: ")
  fmt.Scanln(&dir)

  _, err := os.Stat(dir)
  for (err != nil) {
    if os.IsNotExist(err) {
        log.Print(err)
        fmt.Printf("Der Speicherort %s existiert nicht.\n", dir)
        fmt.Println("Bitte gibt einen gültigen Speicherort an")
    } else {
        fmt.Println("Es ist ein Fehler aufgetreten.")
        fmt.Println("Bist du sicher, dass du Zugriffsberechtigungn auf diesen Ordner hast?")
    }
    fmt.Printf("Speicherort: ")
    fmt.Scanln(&dir)
    _, err = os.Stat(dir)
  }
  db.set_setting("dir", dir)

  db.set_setting("setup", "1")
  fmt.Println("Setup abgeschlossen")
}

func login(username string, password string) bool {
  request("get", baseUrl + homePath, client, nil)
  v := url.Values{}
  v.Set("login", "> Login")
  v.Set("p", password)
  v.Set("u", username)

  var resp string
  // request("post", baseUrl + loginPath, client, v)
  resp, _, _ = request("post", baseUrl + loginPath, client, v)

  re := regexp.MustCompile("timeTable\\.asp")
  match := re.FindStringIndex(resp);
  if match == nil {
    return false
  } else  {
    return true
  }
}

func calendar() {
  s, err := db.get_setting("setup")
  if s != "1" || err == sql.ErrNoRows {
    fmt.Println("Please call setup first")
    os.Exit(0)
  }
  username, err := db.get_encrypted_setting(e, "username")
  if(err != nil) {
    log.Fatal(err)
  }
  password, err := db.get_encrypted_setting(e, "password")
  if(err != nil) {
    log.Fatal(err)
  }

  if !login(string(username), string(password)) {
    log.Fatal("Fehler beim anmelden")
  }


  var resp string
  startDate := time.Now()
  startDate = startDate.AddDate(0, 0, -7)
  endDate := time.Now()
  endDate = endDate.AddDate(0, 6, 0)
  v := url.Values{}
  v.Set("startdt", startDate.Format("02.01.2006"))
  v.Set("enddt", endDate.Format("02.01.2006"))

  resp, _, _ = request("get", baseUrl + calPath, client, v)

  dir, _ := db.get_setting("dir")

  // open output file
  fo, err := os.Create(dir+"/"+username+".ics")
  if err != nil {
      log.Fatal(err)
  }
  // close fo on exit and check for its returned error
  defer func() {
      if err := fo.Close(); err != nil {
          log.Fatal(err)
      }
  }()
  processData(fo, resp)
}


func processData(fo *os.File, resp string) {
  w := bufio.NewWriter(fo)

  var address Room
  var err error
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
              address, err = db.get_address(room)

              if err == sql.ErrNoRows {
                address = crawl_address(room)
                db.set_address(room, address)
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




