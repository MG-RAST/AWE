package shock

import (
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/golib/httpclient"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// TODO use Token

var WAIT_SLEEP = 30 * time.Second
var WAIT_TIMEOUT = time.Duration(-1) * time.Hour
var SHOCK_TIMEOUT = 60 * time.Second
var DATA_SUFFIX = "?download"

type ShockClient struct {
	Host  string
	Token string
	Debug bool
}

type ShockResponse struct {
	Code int       `bson:"status" json:"status"`
	Data ShockNode `bson:"data" json:"data"`
	Errs []string  `bson:"error" json:"error"`
}

type ShockResponseGeneric struct {
	Code int         `bson:"status" json:"status"`
	Data interface{} `bson:"data" json:"data"`
	Errs []string    `bson:"error" json:"error"`
}

type ShockNode struct {
	Id         string             `bson:"id" json:"id"`
	Version    string             `bson:"version" json:"version"`
	File       shockFile          `bson:"file" json:"file"`
	Attributes interface{}        `bson:"attributes" json:"attributes"`
	Indexes    map[string]IdxInfo `bson:"indexes" json:"indexes"`
	//Acl      Acl                 `bson:"acl" json:"-"`
	VersionParts map[string]string `bson:"version_parts" json:"version_parts"`
	Tags         []string          `bson:"tags" json:"tags"`
	//Revisions  []ShockNode       `bson:"revisions" json:"-"`
	Linkages     []linkage `bson:"linkage" json:"linkage"`
	Priority     int       `bson:"priority" json:"priority"`
	CreatedOn    time.Time `bson:"created_on" json:"created_on"`
	LastModified time.Time `bson:"last_modified" json:"last_modified"`
	Expiration   time.Time `bson:"expiration" json:"expiration"`
	Type         string    `bson:"type" json:"type"`
	//Subset     Subset    `bson:"subset" json:"-"`
	Parts *partsList `bson:"parts" json:"parts"`
}

type shockFile struct {
	Name     string            `bson:"name" json:"name"`
	Size     int64             `bson:"size" json:"size"`
	Checksum map[string]string `bson:"checksum" json:"checksum"`
	Format   string            `bson:"format" json:"format"`
	//Path       string        `bson:"path" json:"-"`
	Virtual      bool      `bson:"virtual" json:"virtual"`
	VirtualParts []string  `bson:"virtual_parts" json:"virtual_parts"`
	CreatedOn    time.Time `bson:"created_on" json:"created_on"`
	Locked       *lockInfo `bson:"-" json:"locked"`
}

type IdxInfo struct {
	//Type      string `bson:"index_type" json:"-"`
	TotalUnits  int64 `bson:"total_units" json:"total_units" mapstructure:"total_units"`
	AvgUnitSize int64 `bson:"average_unit_size" json:"average_unit_size" mapstructure:"average_unit_size"`
	//Format  string    `bson:"format" json:"-"`
	CreatedOn time.Time `bson:"created_on" json:"created_on" mapstructure:"created_on"`
	Locked    *lockInfo `bson:"-" json:"locked"`
}

type linkage struct {
	Type      string   `bson: "relation" json:"relation"`
	Ids       []string `bson:"ids" json:"ids"`
	Operation string   `bson:"operation" json:"operation"`
}

type lockInfo struct {
	CreatedOn time.Time `bson:"-" json:"created_on"`
	Error     string    `bson:"-" json:"error"`
}

type partsFile []string

type partsList struct {
	Count       int         `bson:"count" json:"count"`
	Length      int         `bson:"length" json:"length"`
	VarLen      bool        `bson:"varlen" json:"varlen"`
	Parts       []partsFile `bson:"parts" json:"parts"`
	Compression string      `bson:"compression" json:"compression"`
}

type ShockQueryResponse struct {
	Code       int         `bson:"status" json:"status"`
	Data       []ShockNode `bson:"data" json:"data"`
	Errs       []string    `bson:"error" json:"error"`
	Limit      int         `bson:"limit" json:"limit"`
	Offset     int         `bson:"offset" json:"offset"`
	TotalCount int         `bson:"total_count" json:"total_count"`
}

type Opts map[string]string

func (o *Opts) HasKey(key string) bool {
	if _, has := (*o)[key]; has {
		return true
	}
	return false
}

func (o *Opts) Value(key string) string {
	val, _ := (*o)[key]
	return val
}

// *** low-level functions ***

func (sc *ShockClient) Post_request(resource string, query url.Values, header *httpclient.Header, response interface{}) (err error) {
	return sc.Do_request("POST", resource, query, header, response)
}

func (sc *ShockClient) Get_request(resource string, query url.Values, response interface{}) (err error) {
	return sc.Do_request("GET", resource, query, nil, response)
}

func (sc *ShockClient) Put_request(resource string, query url.Values, response interface{}) (err error) {
	return sc.Do_request("PUT", resource, query, nil, response)
}

func (sc *ShockClient) Do_request(method string, resource string, query url.Values, header *httpclient.Header, response interface{}) (err error) {

	//logger.Debug(1, fmt.Sprint("string_url: ", sc.Host))

	var myurl *url.URL
	myurl, err = url.ParseRequestURI(sc.Host)
	if err != nil {
		return
	}

	(*myurl).Path = resource
	(*myurl).RawQuery = query.Encode()

	shockurl := myurl.String()

	logger.Debug(1, fmt.Sprint("shock request url: ", shockurl))
	if sc.Debug {
		fmt.Fprintf(os.Stdout, "Get_request url: %s\n", shockurl)
	}

	if len(shockurl) < 5 {
		err = errors.New("(Do_request) could not parse shockurl: " + shockurl)
		return
	}

	var user *httpclient.Auth
	if sc.Token != "" {
		user = httpclient.GetUserByTokenAuth(sc.Token)
	}

	if header == nil {
		header = &httpclient.Header{}
	}

	var res *http.Response
	res, err = httpclient.Do(method, shockurl, *header, nil, user)
	if err != nil {
		return
	}
	defer res.Body.Close()

	var jsonstream []byte
	jsonstream, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	if sc.Debug {
		fmt.Fprintf(os.Stdout, "json response:\n %s\n", string(jsonstream))
	}

	err = json.Unmarshal(jsonstream, response)

	return
}

// *** high-level functions ***

func (sc *ShockClient) CreateOrUpdate(opts Opts, nodeid string, nodeattr map[string]interface{}) (node *ShockNode, err error) {
	if sc.Debug {
		logger.Debug(1, "(CreateOrUpdate) start")
		defer logger.Debug(1, "(CreateOrUpdate) start")
	}
	host := sc.Host
	token := sc.Token

	if host == "" {
		err = errors.New("(createOrUpdate) host is not defined in Shock node")
		return
	}

	url := host + "/node"
	method := "POST"
	if nodeid != "" {
		url += "/" + nodeid
		method = "PUT"
	}
	form := httpclient.NewForm()
	if opts.HasKey("attributes") {
		form.AddFile("attributes", opts.Value("attributes"))
	}

	if len(nodeattr) != 0 {
		nodeattr_json, jerr := json.Marshal(nodeattr)
		if jerr != nil {
			err = errors.New("(CreateOrUpdate) error marshalling NodeAttr")
			return
		}
		form.AddParam("attributes_str", string(nodeattr_json[:]))
	}

	var uploadType string
	if opts.HasKey("upload_type") {
		uploadType = opts.Value("upload_type")
	}

	if uploadType != "" {
		switch uploadType {
		case "basic":
			if opts.HasKey("file") { // upload_type: basic , file=...
				form.AddFile("upload", opts.Value("file"))
			}
		case "parts":
			if opts.HasKey("parts") {
				form.AddParam("parts", opts.Value("parts"))
			} else {
				err = errors.New("(CreateOrUpdate) (case:parts) missing partial upload parameter: parts")
				return
			}
			if opts.HasKey("file_name") {
				form.AddParam("file_name", opts.Value("file_name"))
			}
		case "part":
			if opts.HasKey("part") && opts.HasKey("file") {
				form.AddFile(opts.Value("part"), opts.Value("file"))
			} else {
				err = errors.New("(CreateOrUpdate) (case:part) missing partial upload parameter: part or file")
				return
			}
		case "remote_path":
			if opts.HasKey("remote_path") {
				form.AddParam("path", opts.Value("remote_path"))
			} else {
				err = errors.New("(CreateOrUpdate) (case:remote_path) missing remote path parameter: path")
				return
			}
		case "virtual_file":
			if opts.HasKey("virtual_file") {
				form.AddParam("type", "virtual")
				form.AddParam("source", opts.Value("virtual_file"))
			} else {
				err = errors.New("(CreateOrUpdate) (case:virtual_file) missing virtual node parameter: source")
				return
			}
		case "index":
			if opts.HasKey("index_type") {
				url += "/index/" + opts.Value("index_type")
			} else {
				err = errors.New("(CreateOrUpdate) (case:index) missing index type when creating index")
				return
			}
		case "copy":
			if opts.HasKey("parent_node") {
				form.AddParam("copy_data", opts.Value("parent_node"))
			} else {
				err = errors.New("(CreateOrUpdate) (case:copy) missing copy node parameter: parent_node")
				return
			}
			if opts.HasKey("copy_indexes") {
				form.AddParam("copy_indexes", "1")
			}
		case "subset":
			if opts.HasKey("parent_node") && opts.HasKey("parent_index") && opts.HasKey("file") {
				form.AddParam("parent_node", opts.Value("parent_node"))
				form.AddParam("parent_index", opts.Value("parent_index"))
				form.AddFile("subset_indices", opts.Value("file"))
			} else {
				err = errors.New("(CreateOrUpdate) (case:subset) missing subset node parameter: parent_node or parent_index or file")
				return
			}
		}
	}

	err = form.Create()
	if err != nil {
		return
	}

	headers := httpclient.Header{
		"Content-Type":   []string{form.ContentType},
		"Content-Length": []string{strconv.FormatInt(form.Length, 10)},
	}
	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}
	if sc.Debug {
		fmt.Printf("url: %s %s\n", method, url)
	}
	var res *http.Response
	res, err = httpclient.Do(method, url, headers, form.Reader, user)
	if err != nil {
		return
	}

	defer res.Body.Close()
	jsonstream, _ := ioutil.ReadAll(res.Body)
	response := new(ShockResponse)
	if err = json.Unmarshal(jsonstream, response); err != nil {
		err = fmt.Errorf("(CreateOrUpdate) (httpclient.Do) failed to marshal response:\"%s\"", jsonstream)
		return
	}
	if len(response.Errs) > 0 {
		err = fmt.Errorf("(CreateOrUpdate) type=%s, method=%s, url=%s, error=%s", uploadType, method, url, strings.Join(response.Errs, ","))
		return
	}
	node = &response.Data

	return
}

func (sc *ShockClient) ShockPutIndex(nodeid string, indexname string) (err error) {
	if indexname == "" {
		return
	}
	opts := Opts{}
	opts["upload_type"] = "index"
	opts["index_type"] = indexname
	logger.Debug(2, fmt.Sprintf("(ShockPutIndex) node=%s/node/%s index=%s", sc.Host, nodeid, indexname))
	_, err = sc.CreateOrUpdate(opts, nodeid, nil)
	return
}

func (sc *ShockClient) WaitIndex(nodeid string, indexname string) (index IdxInfo, err error) {
	if indexname == "" {
		return
	}
	var node *ShockNode
	var has bool
	startTime := time.Now()
	for {
		node, err = sc.Get_node(nodeid)
		if err != nil {
			return
		}
		index, has = node.Indexes[indexname]
		if !has {
			err = fmt.Errorf("(shock.WaitIndex) index does not exist: node=%s, index=%s", nodeid, indexname)
			return
		}
		if (index.Locked != nil) && (index.Locked.Error != "") {
			// we have an error, return it
			err = fmt.Errorf("(shock.WaitIndex) error creating index: node=%s, index=%s, error=%s", nodeid, indexname, index.Locked.Error)
			return
		}
		if index.Locked == nil {
			// no longer locked
			return
		}
		// need a resonable timeout
		currTime := time.Now()
		expireTime := currTime.Add(WAIT_TIMEOUT) // adding negative time to current time, because no subtraction function
		if startTime.Before(expireTime) {
			err = fmt.Errorf("(shock.WaitIndex) timeout waiting on index lock: node=%s, index=%s", nodeid, indexname)
			return
		}
		time.Sleep(WAIT_SLEEP)
	}
	return
}

func (sc *ShockClient) WaitFile(nodeid string) (node *ShockNode, err error) {
	startTime := time.Now()
	for {
		node, err = sc.Get_node(nodeid)
		if err != nil {
			return
		}
		if (node.File.Locked != nil) && (node.File.Locked.Error != "") {
			// we have an error, return it
			err = fmt.Errorf("(shock.WaitFile) error waiting on file lock: node=%s, error=%s", nodeid, node.File.Locked.Error)
			return
		}
		if node.File.Locked == nil {
			// no longer locked
			return
		}
		// need a resonable timeout
		currTime := time.Now()
		expireTime := currTime.Add(WAIT_TIMEOUT) // adding negative time to current time, because no subtraction function
		if startTime.Before(expireTime) {
			err = fmt.Errorf("(shock.WaitFile) timeout waiting on file lock: node=%s", nodeid)
			return
		}
		time.Sleep(WAIT_SLEEP)
	}
	return
}

// should return node id if you create new node (i.e. do not specify node id)
func (sc *ShockClient) PutFileToShock(filename string, nodeid string, rank int, attrfile string, ntype string, formopts map[string]string, nodeattr map[string]interface{}) (new_node_id string, err error) {

	opts := Opts{}
	fi, _ := os.Stat(filename)
	if (attrfile != "") && (rank < 2) {
		opts["attributes"] = attrfile
	}
	if filename != "" {
		opts["file"] = filename
	}
	if rank == 0 {
		opts["upload_type"] = "basic"
	} else {
		opts["upload_type"] = "part"
		opts["part"] = strconv.Itoa(rank)
	}
	if (ntype == "subset") && (rank == 0) && (fi.Size() == 0) {
		opts["upload_type"] = "basic"
	} else if ((ntype == "copy") || (ntype == "subset")) && (len(formopts) > 0) {
		opts["upload_type"] = ntype
		for k, v := range formopts {
			opts[k] = v
		}
	}

	var node *ShockNode
	node, err = sc.CreateOrUpdate(opts, nodeid, nodeattr)
	if err != nil {
		err = fmt.Errorf("(PutFileToShock) (CreateOrUpdate) failed (%s): %v", sc.Host, err)
		return
	}
	if node != nil {
		new_node_id = node.Id
	}

	return
}

func (sc *ShockClient) PostNodeWithToken(filename string, numParts int) (nodeid string, err error) {
	opts := Opts{}
	var node *ShockNode

	node, err = sc.CreateOrUpdate(opts, "", nil)
	if err != nil {
		err = fmt.Errorf("(PostNodeWithToken) (CreateOrUpdate) failed (%s): %v", sc.Host, err)
		return
	}
	//create "parts" for output splits
	if numParts > 1 {
		opts["upload_type"] = "parts"
		opts["file_name"] = filename
		opts["parts"] = strconv.Itoa(numParts)
		_, err = sc.CreateOrUpdate(opts, node.Id, nil)
		if err != nil {
			nodeid = node.Id
			err = fmt.Errorf("(PostNodeWithToken) (CreateOrUpdate) failed (%s, %s): %v", sc.Host, node.Id, err)
			return
		}
	}
	return node.Id, nil
}

func (sc *ShockClient) Get_node_download_url(node ShockNode) (download_url string, err error) {

	var myurl *url.URL
	myurl, err = url.ParseRequestURI(sc.Host)
	if err != nil {
		return
	}
	(*myurl).Path = fmt.Sprint("node/", node.Id)
	(*myurl).RawQuery = "download"

	download_url = myurl.String()
	return
}

func (sc *ShockClient) Make_public(node_id string) (sqr_p *ShockResponseGeneric, err error) {
	sqr_p = new(ShockResponseGeneric)
	err = sc.Put_request("/node/"+node_id+"/acl/public_read", nil, &sqr_p)
	return
}

// example: query_response_p, err := sc.Shock_query(host, url.Values{"docker": {"1"}, "docker_image_name" : {"wgerlach/bowtie2:2.2.0"}});
func (sc *ShockClient) Query(query url.Values) (sqr_p *ShockQueryResponse, err error) {
	query.Add("query", "")
	sqr_p = new(ShockQueryResponse)
	err = sc.Get_request("/node/", query, &sqr_p)
	return
}

func (sc *ShockClient) Get_node(node_id string) (node *ShockNode, err error) {
	sqr_p := new(ShockResponse)
	err = sc.Get_request("/node/"+node_id, nil, &sqr_p)

	if len(sqr_p.Errs) > 0 {
		err = fmt.Errorf("(Get_node) node=%s: %s", node_id, strings.Join(sqr_p.Errs, ","))
		return
	}

	node = &sqr_p.Data
	if node == nil {
		err = fmt.Errorf("(Get_node) node=%s: empty node returned from Shock", node_id)
	}
	return
}

// old-style functions that probably should to be refactored

func ShockGet(host string, nodeid string, token string) (node *ShockNode, err error) {
	if host == "" || nodeid == "" {
		err = errors.New("(ShockGet) empty shock host or node id")
		return
	}
	logger.Debug(3, fmt.Sprintf("(ShockGet) %s %s %s", host, nodeid, token))

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
	case <-time.After(SHOCK_TIMEOUT):
		err = errors.New("(ShockGet) timeout when getting node from shock, url=" + shockurl)
		return
	}

	if err != nil {
		return
	}
	defer res.Body.Close()

	var jsonstream []byte
	jsonstream, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	response := new(ShockResponse)
	if err = json.Unmarshal(jsonstream, response); err != nil {
		return
	}
	if len(response.Errs) > 0 {
		err = fmt.Errorf("(ShockGet) (response) %s", strings.Join(response.Errs, ","))
		return
	}
	node = &response.Data
	if node == nil {
		err = errors.New("(ShockGet) (response) empty node got from Shock")
	}
	return
}

func ShockDelete(host string, nodeid string, token string) (err error) {
	if host == "" || nodeid == "" {
		err = errors.New("(ShockDelete) empty shock host or node id")
		return
	}

	var res *http.Response
	shockurl := fmt.Sprintf("%s/node/%s", host, nodeid)

	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}

	c := make(chan int, 1)
	go func() {
		res, err = httpclient.Delete(shockurl, httpclient.Header{}, nil, user)
		c <- 1 //we are ending
	}()
	select {
	case <-c:
	//go ahead
	case <-time.After(SHOCK_TIMEOUT):
		err = errors.New("(ShockDelete) timeout when getting node from shock, url=" + shockurl)
		return
	}
	if err != nil {
		return
	}
	defer res.Body.Close()

	var jsonstream []byte
	jsonstream, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	response := new(ShockResponse)
	if err = json.Unmarshal(jsonstream, response); err != nil {
		return
	}
	if len(response.Errs) > 0 {
		err = fmt.Errorf("(ShockDelete) (response) %s", strings.Join(response.Errs, ","))
		return
	}
	return
}

//fetch file by shock url
func FetchFile(filename string, url string, token string, uncompress string, computeMD5 bool) (size int64, md5sum string, err error) {
	logger.Debug(1, "(FetchFile) fetching file name=%s, url=%s\n", filename, url)

	var localfile *os.File
	localfile, err = os.Create(filename)
	if err != nil {
		return
	}
	defer localfile.Close()

	var body io.ReadCloser
	body, err = FetchShockStream(url, token)
	if err != nil {
		err = errors.New("(FetchFile) " + err.Error())
		return
	}
	defer body.Close()

	// set md5 compute
	md5h := md5.New()

	if uncompress == "" {
		//logger.Debug(1, fmt.Sprintf("downloading file %s from %s", filename, url))
		// split stream to file and md5
		var dst io.Writer
		if computeMD5 {
			dst = io.MultiWriter(localfile, md5h)
		} else {
			dst = localfile
		}
		size, err = io.Copy(dst, body)
		if err != nil {
			return
		}
	} else if uncompress == "gzip" {
		//logger.Debug(1, fmt.Sprintf("downloading and unzipping file %s from %s", filename, url))
		// split stream to gzip and md5
		var input io.ReadCloser
		if computeMD5 {
			pReader, pWriter := io.Pipe()
			defer pReader.Close()
			dst := io.MultiWriter(pWriter, md5h)
			go func() {
				io.Copy(dst, body)
				pWriter.Close()
			}()
			input = pReader
		} else {
			input = body
		}

		gr, gerr := gzip.NewReader(input)
		if gerr != nil {
			err = gerr
			return
		}
		defer gr.Close()
		size, err = io.Copy(localfile, gr)
		if err != nil {
			return
		}
	} else {
		err = errors.New("(FetchFile) uncompress method unknown: " + uncompress)
		return
	}

	if computeMD5 {
		md5sum = fmt.Sprintf("%x", md5h.Sum(nil))
	}
	return
}

func FetchShockStream(url string, token string) (r io.ReadCloser, err error) {

	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}

	//download file from Shock
	var res *http.Response
	res, err = httpclient.Get(url, httpclient.Header{}, nil, user)
	if err != nil {
		err = errors.New("(FetchShockStream) httpclient.Get returned: " + err.Error())
		return
	}

	if res.StatusCode != 200 { //err in fetching data
		resbody, _ := ioutil.ReadAll(res.Body)
		err = errors.New(fmt.Sprintf("(FetchShockStream) url=%s, res=%s", url, resbody))
		return
	}

	return res.Body, err
}

// source:  http://stackoverflow.com/a/22259280
// TODO this is not shock related, need another package
func CopyFile(src, dst string) (size int64, err error) {
	var src_file *os.File
	src_file, err = os.Open(src)
	if err != nil {
		return
	}
	defer src_file.Close()

	var src_file_stat os.FileInfo
	src_file_stat, err = src_file.Stat()
	if err != nil {
		return
	}

	if !src_file_stat.Mode().IsRegular() {
		err = fmt.Errorf("%s is not a regular file", src)
		return
	}

	var dst_file *os.File
	dst_file, err = os.Create(dst)
	if err != nil {
		return
	}
	defer dst_file.Close()
	size, err = io.Copy(dst_file, src_file)
	return
}
