// this package contains modified code based on following github repo:
// https://github.com/jaredwilkening/httpclient
package httpclient

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
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
	return DoTimeout(t, url, header, data, user, time.Second*0) // TODO decrease to 60 seconds if SHOCK issue withchunks is solved
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

func GetTimeout(url string, header Header, data io.Reader, user *Auth, ReadWriteTimeout time.Duration) (resp *http.Response, err error) {
	return DoTimeout("GET", url, header, data, user, ReadWriteTimeout)
}

func DoTimeout(t string, url string, header Header, data io.Reader, user *Auth, ReadWriteTimeout time.Duration) (*http.Response, error) {
	trans := newTransport()

	ConnectTimeout := time.Second * 10

	if ReadWriteTimeout != 0 {

		trans.Dial = func(netw, addr string) (net.Conn, error) {
			c, err := net.DialTimeout(netw, addr, ConnectTimeout)
			if err != nil {
				//return nil, err
				return nil, errors.New("net.DialTimeout complains: " + err.Error())
			}
			if ReadWriteTimeout > 0 {
				timeoutConn := &rwTimeoutConn{
					TCPConn:   c.(*net.TCPConn),
					rwTimeout: ReadWriteTimeout,
				}
				return timeoutConn, nil
			}
			return c, nil
		}

	}

	trans.DisableKeepAlives = true
	req, err := http.NewRequest(t, url, data)
	if err != nil {
		return nil, errors.New("http.NewRequest complains: " + err.Error())
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

// A net.Conn that sets a deadline for every Read or Write operation
type rwTimeoutConn struct {
	*net.TCPConn
	rwTimeout time.Duration
}

func (c *rwTimeoutConn) Read(b []byte) (int, error) {
	err := c.TCPConn.SetReadDeadline(time.Now().Add(c.rwTimeout))
	if err != nil {
		//return 0, err
		return 0, errors.New("c.TCPConn.SetReadDeadline complains: " + err.Error())
	}
	return c.TCPConn.Read(b)
}
func (c *rwTimeoutConn) Write(b []byte) (int, error) {
	err := c.TCPConn.SetWriteDeadline(time.Now().Add(c.rwTimeout))
	if err != nil {
		//return 0, err
		return 0, errors.New("c.TCPConn.SetWriteDeadline complains: " + err.Error())
	}
	return c.TCPConn.Write(b)
}
