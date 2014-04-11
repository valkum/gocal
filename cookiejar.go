package main

import (
    "net/http"
    "net/url"
    "time"
    "strconv"
    "strings"
    // spew "github.com/davecgh/go-spew/spew"
)


/**
 * As we use hardcoded URL we don't need a secure CookieJar implementation
 *
 */
type myCookieJar struct {
    cookies map[string]*http.Cookie
}

func (c *myCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
    if c.cookies == nil {
        c.cookies = make(map[string]*http.Cookie, 0)
    }

    for _, it := range cookies {
            c.cookies[it.Name] = it
    }
}

func (c *myCookieJar) Cookies(u *url.URL) (cookies []*http.Cookie) {
    // spew.Dump(c)
    for _, e := range c.cookies {
        cookies = append(cookies, &http.Cookie{Name: e.Name, Value: e.Value})
    }
    return cookies
}


// readSetCookies parses all "Set-Cookie" values from
// the header h and returns the successfully parsed Cookies.
func readSetCookies(h http.Header) []*http.Cookie {
    cookies := []*http.Cookie{}
    for _, line := range h["Set-Cookie"] {
        parts := strings.Split(strings.TrimSpace(line), ";")
        if len(parts) == 1 && parts[0] == "" {
            continue
        }
        parts[0] = strings.TrimSpace(parts[0])
        j := strings.Index(parts[0], "=")
        if j < 0 {
            continue
        }
        name, value := parts[0][:j], parts[0][j+1:]
        if !isCookieNameValid(name) {
            continue
        }
        value, success := parseCookieValue(value)
        if !success {
            continue
        }
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
