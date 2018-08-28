package httpclient

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type RestClient struct {
	User   *Auth
	Header Header
	Page   *pagination
}

type pagination struct {
	url        string
	currPos    int
	offsetName string
	pageName   string
	currPage   []interface{}
}

type PageItem struct {
	Data interface{}
}

func (rc *RestClient) Get(url string) (jsonstream []byte, err error) {
	var resp *http.Response
	resp, err = Get(url, rc.Header, rc.User)
	if err != nil {
		return
	}
	jsonstream, err = ioutil.ReadAll(resp.Body)
	return
}

func (rc *RestClient) Delete(url string) (jsonstream []byte, err error) {
	var resp *http.Response
	resp, err = Delete(url, rc.Header, rc.User)
	if err != nil {
		return
	}
	jsonstream, err = ioutil.ReadAll(resp.Body)
	return
}

func (rc *RestClient) Post(url string, data io.Reader) (jsonstream []byte, err error) {
	var resp *http.Response
	resp, err = Post(url, rc.Header, data, rc.User)
	if err != nil {
		return
	}
	jsonstream, err = ioutil.ReadAll(resp.Body)
	return
}

func (rc *RestClient) Put(url string, data io.Reader) (jsonstream []byte, err error) {
	var resp *http.Response
	resp, err = Put(url, rc.Header, data, rc.User)
	if err != nil {
		return
	}
	jsonstream, err = ioutil.ReadAll(resp.Body)
	return
}

func (rc *RestClient) InitPagination(url string, pageName string, offsetName string) {
	rc.Page = &pagination{
		url:        url,
		currPos:    0,
		offsetName: offsetName,
		pageName:   pageName,
		currPage:   []interface{}{},
	}
}

// if error is io.EOF, than no more items left
func (rc *RestClient) Next() (item *PageItem, err error) {
	// pagination not initialized
	if rc.Page == nil {
		err = errors.New("InitPagination is required before retreiving items")
		return
	}
	// at end of current page, request new one
	if (len(rc.Page.currPage) == 0) || (len(rc.Page.currPage) == (rc.Page.currPos + 1)) {
		rc.Page.nextPage(rc)
		// no more pages left to retrieve
		if len(rc.Page.currPage) == 0 {
			err = io.EOF
			return
		}
	} else {
		// get next position
		rc.Page.currPos += 1
	}
	item = &PageItem{Data: rc.Page.currPage[rc.Page.currPos]}
	return
}

// get next page, resets CurrPage and CurrPos
func (p *pagination) nextPage(rc *RestClient) {
	p.updateOffset()

	result, err := rc.Get(p.url)
	if err != nil {
		p.currPage = []interface{}{}
		return
	}

	var object map[string]interface{}
	json.Unmarshal(result, &object)

	p.currPage = object[p.pageName].([]interface{})
	p.currPos = 0
}

func (p *pagination) updateOffset() {
	myurl, _ := url.Parse(p.url)
	values := myurl.Query()

	offset, _ := strconv.Atoi(values.Get(p.offsetName))
	newOffset := offset + len(p.currPage)

	values.Set(p.offsetName, strconv.Itoa(newOffset))
	(*myurl).RawQuery = values.Encode()
	p.url = myurl.String()
}
