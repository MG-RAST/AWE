// this package contains modified code based on following github repo:
// https://github.com/jaredwilkening/httpclient
package httpclient

import (
	"crypto/tls"
	"io"
	"net/http"
	"strings"
)

type Header http.Header

type Auth struct {
	Type     string
	Username string
	Password string
	Token    string
}

func newTransport() *http.Transport {
	return &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
}

// support multiple token types with datatoken
// backwards compatable, if no type given default to OAuth
func GetUserByTokenAuth(token string) (user *Auth) {
	tmp := strings.Split(token, " ")
	if len(tmp) > 1 {
		user = &Auth{Type: tmp[0], Token: tmp[1]}
	} else {
		user = &Auth{Type: "OAuth", Token: token}
	}
	return
}

func GetUserByBasicAuth(username, password string) (user *Auth) {
	user = &Auth{Type: "basic", Username: username, Password: password}
	return
}

func Do(t string, url string, header Header, data io.Reader, user *Auth) (*http.Response, error) {
	trans := newTransport()
	trans.DisableKeepAlives = true
	req, err := http.NewRequest(t, url, data)
	if err != nil {
		return nil, err
	}
	if user != nil {
		if user.Type == "basic" {
			req.SetBasicAuth(user.Username, user.Password)
		} else {
			req.Header.Add("Authorization", user.Type+" "+user.Token)
		}
	}
	for k, v := range header {
		for _, v2 := range v {
			req.Header.Add(k, v2)
		}
	}
	return trans.RoundTrip(req)
}

func Get(url string, header Header, data io.Reader, user *Auth) (resp *http.Response, err error) {
	return Do("GET", url, header, data, user)
}

func Post(url string, header Header, data io.Reader, user *Auth) (resp *http.Response, err error) {
	return Do("POST", url, header, data, user)
}

func Put(url string, header Header, data io.Reader, user *Auth) (resp *http.Response, err error) {
	return Do("PUT", url, header, data, user)
}

func Delete(url string, header Header, data io.Reader, user *Auth) (resp *http.Response, err error) {
	return Do("DELETE", url, header, data, user)
}
