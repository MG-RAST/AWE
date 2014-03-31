package core

import (
	//"github.com/MG-RAST/AWE/lib/httpclient"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/httpclient"
	"github.com/MG-RAST/AWE/lib/logger"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TODO use Token
type ShockClient struct {
	Host  string
	Token string
}

type ShockResponse struct {
	Code int       `bson:"status" json:"status"`
	Data ShockNode `bson:"data" json:"data"`
	Errs []string  `bson:"error" json:"error"`
}

type ShockNode struct {
	Id         string             `bson:"id" json:"id"`
	Version    string             `bson:"version" json:"version"`
	File       shockfile          `bson:"file" json:"file"`
	Attributes interface{}        `bson:"attributes" json:"attributes"`
	Indexes    map[string]IdxInfo `bson:"indexes" json:"indexes"`
	//Acl          Acl                `bson:"acl" json:"-"`
	VersionParts map[string]string `bson:"version_parts" json:"-"`
	Tags         []string          `bson:"tags" json:"tags"`
	//	Revisions    []ShockNode       `bson:"revisions" json:"-"`
	Linkages []linkage `bson:"linkage" json:"linkages"`
}

type shockfile struct {
	Name         string            `bson:"name" json:"name"`
	Size         int64             `bson:"size" json:"size"`
	Checksum     map[string]string `bson:"checksum" json:"checksum"`
	Format       string            `bson:"format" json:"format"`
	Path         string            `bson:"path" json:"-"`
	Virtual      bool              `bson:"virtual" json:"virtual"`
	VirtualParts []string          `bson:"virtual_parts" json:"virtual_parts"`
}

type IdxInfo struct {
	Type        string `bson:"index_type" json:"-"`
	TotalUnits  int64  `bson:"total_units" json:"total_units"`
	AvgUnitSize int64  `bson:"average_unit_size" json:"average_unit_size"`
}

type linkage struct {
	Type      string   `bson: "relation" json:"relation"`
	Ids       []string `bson:"ids" json:"ids"`
	Operation string   `bson:"operation" json:"operation"`
}

type ShockQueryResponse struct {
	Code       int         `bson:"status" json:"status"`
	Data       []ShockNode `bson:"data" json:"data"`
	Errs       []string    `bson:"error" json:"error"`
	Limit      int         `bson:"limit" json:"limit"`
	Offset     int         `bson:"offset" json:"offset"`
	TotalCount int         `bson:"total_count" json:"total_count"`
}

func ShockGet(host string, nodeid string, token string) (node *ShockNode, err error) {
	if host == "" || nodeid == "" {
		return nil, errors.New("empty shock host or node id")
	}

	var res *http.Response
	shockurl := fmt.Sprintf("%s/node/%s", host, nodeid)

	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}

	c := make(chan int, 1)
	go func() {
		res, err = httpclient.Get(shockurl, httpclient.Header{}, nil, user)
		c <- 1 //we are ending
	}()
	select {
	case <-c:
	//go ahead
	case <-time.After(conf.SHOCK_TIMEOUT):
		return nil, errors.New("timeout when getting node from shock, url=" + shockurl)
	}
	if err != nil {
		return
	}
	defer res.Body.Close()

	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	response := new(ShockResponse)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return nil, err
	}
	if len(response.Errs) > 0 {
		return nil, errors.New(strings.Join(response.Errs, ","))
	}
	node = &response.Data
	if node == nil {
		err = errors.New("empty node got from Shock")
	}
	return
}

func (sc *ShockClient) Get_node_download_url(node ShockNode) (download_url string, err error) {
	myurl, err := url.ParseRequestURI(sc.Host)
	if err != nil {
		return "", err
	}
	(*myurl).Path = fmt.Sprint("node/", node.Id)
	(*myurl).RawQuery = "download"

	download_url = myurl.String()
	return
}

// example: query_response_p, err := sc.Shock_query(host, url.Values{"docker": {"1"}, "docker_image_name" : {"wgerlach/bowtie2:2.2.0"}});
func (sc *ShockClient) Query(query url.Values) (sqr_p *ShockQueryResponse, err error) {

	query.Add("query", "")

	sqr_p = new(ShockQueryResponse)
	err = sc.Get_request("/node/", query, &sqr_p)

	return
}

func (sc *ShockClient) Get_request(resource string, query url.Values, response interface{}) (err error) {

	logger.Debug(1, fmt.Sprint("string_url: ", sc.Host))

	myurl, err := url.ParseRequestURI(sc.Host)
	if err != nil {
		return err
	}

	(*myurl).Path = resource
	(*myurl).RawQuery = query.Encode()

	shockurl := myurl.String()

	logger.Debug(1, fmt.Sprint("shock request url: ", shockurl))

	if len(shockurl) < 5 {
		return errors.New("could not parse SHOCK_DOCKER_IMAGE_REPOSITORY")
	}

	var res *http.Response

	c := make(chan int, 1)
	go func() {
		res, err = httpclient.Get(shockurl, httpclient.Header{}, nil, nil)
		c <- 1 //we are ending
	}()
	select {
	case <-c:
	//go ahead
	case <-time.After(conf.SHOCK_TIMEOUT):
		return errors.New("timeout when getting node from shock, url=" + shockurl)
	}
	if err != nil {
		return
	}
	defer res.Body.Close()

	jsonstream, err := ioutil.ReadAll(res.Body)
	//logger.Debug(1, string(jsonstream))
	if err != nil {
		return err
	}

	//response := new(result)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return err
	}
	//if len(response.Errs) > 0 {
	//	return errors.New(strings.Join(response.Errs, ","))
	//}
	//node = &response.Data
	//if node == nil {
	//	err = errors.New("empty node got from Shock")
	//}
	return
}
