package shock

import (
	//"github.com/MG-RAST/AWE/lib/httpclient"
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	//"github.com/MG-RAST/AWE/lib/conf"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/golib/httpclient"
)

// TODO use Token

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
	//Acl      Acl                `bson:"acl" json:"-"`
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
}

type IdxInfo struct {
	//Type        string    `bson:"index_type" json:"-"`
	TotalUnits  int64 `bson:"total_units" json:"total_units" mapstructure:"total_units"`
	AvgUnitSize int64 `bson:"average_unit_size" json:"average_unit_size" mapstructure:"average_unit_size"`
	//Format      string    `bson:"format" json:"-"`
	CreatedOn time.Time `bson:"created_on" json:"created_on" mapstructure:"created_on"`
}

type linkage struct {
	Type      string   `bson: "relation" json:"relation"`
	Ids       []string `bson:"ids" json:"ids"`
	Operation string   `bson:"operation" json:"operation"`
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

	myurl, err := url.ParseRequestURI(sc.Host)
	if err != nil {
		return err
	}

	(*myurl).Path = resource
	(*myurl).RawQuery = query.Encode()

	shockurl := myurl.String()

	logger.Debug(1, fmt.Sprint("shock request url: ", shockurl))
	if sc.Debug {
		fmt.Fprintf(os.Stdout, "Get_request url: %s\n", shockurl)
	}

	if len(shockurl) < 5 {
		return errors.New("could not parse shockurl: " + shockurl)
	}

	var user *httpclient.Auth
	if sc.Token != "" {
		user = httpclient.GetUserByTokenAuth(sc.Token)
	}

	var res *http.Response

	if header == nil {
		header = &httpclient.Header{}
	}

	res, err = httpclient.Do(method, shockurl, *header, nil, user)

	if err != nil {
		return
	}
	defer res.Body.Close()

	jsonstream, err := ioutil.ReadAll(res.Body)

	if sc.Debug {
		fmt.Fprintf(os.Stdout, "json response:\n %s\n", string(jsonstream))
	}

	//logger.Debug(1, string(jsonstream))
	if err != nil {
		return err
	}

	//response := new(result)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return err
	}
	return
}

// *** high-level functions ***

func (sc *ShockClient) FetchFile(filename string, url string, uncompress string, computeMD5 bool) (size int64, md5sum string, err error) {
	return FetchFile(filename, url, sc.Token, uncompress, computeMD5)
}

func (sc *ShockClient) CreateOrUpdate(opts Opts, nodeid string, nodeattr map[string]interface{}) (node *ShockNode, err error) {
	if sc.Debug {
		logger.Debug(1, "(CreateOrUpdate) start")
		defer logger.Debug(1, "(CreateOrUpdate) stop")
	}
	host := sc.Host
	token := sc.Token

	if host == "" {
		err = fmt.Errorf("(createOrUpdate) host is not defined in Shock node")
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
		nodeattr_json, err := json.Marshal(nodeattr)
		if err != nil {
			return nil, errors.New("error marshalling NodeAttr")
		}
		form.AddParam("attributes_str", string(nodeattr_json[:]))
	}

	if opts.HasKey("upload_type") {
		switch opts.Value("upload_type") {
		case "basic":
			if opts.HasKey("file") { // upload_type: basic , file=...
				form.AddFile("upload", opts.Value("file"))
			}
		case "parts":
			if opts.HasKey("parts") {
				form.AddParam("parts", opts.Value("parts"))
			} else {
				return nil, errors.New("missing partial upload parameter: parts")
			}
			if opts.HasKey("file_name") {
				form.AddParam("file_name", opts.Value("file_name"))
			}
		case "part":
			if opts.HasKey("part") && opts.HasKey("file") {
				form.AddFile(opts.Value("part"), opts.Value("file"))
			} else {
				return nil, errors.New("missing partial upload parameter: part or file")
			}
		case "remote_path":
			if opts.HasKey("remote_path") {
				form.AddParam("path", opts.Value("remote_path"))
			} else {
				return nil, errors.New("missing remote path parameter: path")
			}
		case "virtual_file":
			if opts.HasKey("virtual_file") {
				form.AddParam("type", "virtual")
				form.AddParam("source", opts.Value("virtual_file"))
			} else {
				return nil, errors.New("missing virtual node parameter: source")
			}
		case "index":
			if opts.HasKey("index_type") {
				url += "/index/" + opts.Value("index_type")
			} else {
				return nil, errors.New("missing index type when creating index")
			}
		case "copy":
			if opts.HasKey("parent_node") {
				form.AddParam("copy_data", opts.Value("parent_node"))
			} else {
				return nil, errors.New("missing copy node parameter: parent_node")
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
				return nil, errors.New("missing subset node parameter: parent_node or parent_index or file")
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
		fmt.Printf("(CreateOrUpdate) url: %s %s\n", method, url)
	}
	if res, err := httpclient.Do(method, url, headers, form.Reader, user); err == nil {
		defer res.Body.Close()
		jsonstream, _ := ioutil.ReadAll(res.Body)
		response := new(ShockResponse)
		if err := json.Unmarshal(jsonstream, response); err != nil {
			return nil, errors.New(fmt.Sprintf("failed to marshal response:\"%s\"", jsonstream))
		}
		if len(response.Errs) > 0 {
			return nil, errors.New(strings.Join(response.Errs, ","))
		}
		node = &response.Data
	} else {
		return nil, err
	}
	return
}

func (sc *ShockClient) ShockPutIndex(nodeid string, indexname string) (err error) {
	host := sc.Host

	opts := Opts{}
	opts["upload_type"] = "index"
	opts["index_type"] = indexname
	logger.Debug(2, fmt.Sprintf("(ShockPutIndex) node=%s/node/%s index=%s", host, nodeid, indexname))
	_, err = sc.CreateOrUpdate(opts, nodeid, nil)
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

	node, err := sc.CreateOrUpdate(opts, nodeid, nodeattr)
	if err != nil {
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
		err = fmt.Errorf("(1) createOrUpdate in PostNodeWithToken failed (%s): %v", sc.Host, err)
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
			err = fmt.Errorf("(2) createOrUpdate in PostNodeWithToken failed (%s, %s): %v", sc.Host, node.Id, err)
			return
		}
	}
	return node.Id, nil
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

//func (sc *ShockClient) Get_request(node_id string) (sqr_p *ShockResponse, err error) {

//	sqr_p = new(ShockResponse)
//	err = sc.Get_request("/node/"+node_id, nil, &sqr_p)

//	return
//}

func (sc *ShockClient) Get_node(node_id string) (node *ShockNode, err error) {

	sqr_p := new(ShockResponse)
	err = sc.Get_request("/node/"+node_id, nil, &sqr_p)

	if len(sqr_p.Errs) > 0 {
		return nil, errors.New(strings.Join(sqr_p.Errs, ","))
	}

	node = &sqr_p.Data
	if node == nil {
		err = errors.New("empty node got from Shock")
	}

	return
}

// old-style functions that probably should to be refactored

func ShockGet(host string, nodeid string, token string) (node *ShockNode, err error) {
	if host == "" || nodeid == "" {
		return nil, errors.New("empty shock host or node id")
	}
	logger.Debug(3, fmt.Sprintf("ShockGet: %s %s %s", host, nodeid, token))

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

func ShockDelete(host string, nodeid string, token string) (err error) {
	if host == "" || nodeid == "" {
		return errors.New("empty shock host or node id")
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
		return errors.New("timeout when getting node from shock, url=" + shockurl)
	}
	if err != nil {
		return err
	}
	defer res.Body.Close()

	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	response := new(ShockResponse)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return err
	}
	if len(response.Errs) > 0 {
		return errors.New(strings.Join(response.Errs, ","))
	}
	return
}

//fetch file by shock url
func FetchFile(filename string, url string, token string, uncompress string, computeMD5 bool) (size int64, md5sum string, err error) {
	logger.Debug(1, "(FetchFile) fetching file name=%s, url=%s\n", filename, url)

	localfile, err := os.Create(filename)
	if err != nil {
		return 0, "", err
	}
	defer localfile.Close()

	body, err := FetchShockStream(url, token)
	if err != nil {
		return 0, "", errors.New("FetchShockStream returned: " + err.Error())
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
			return 0, "", err
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

		gr, err := gzip.NewReader(input)
		if err != nil {
			return 0, "", err
		}
		defer gr.Close()
		size, err = io.Copy(localfile, gr)
		if err != nil {
			return 0, "", err
		}
	} else {
		return 0, "", errors.New("uncompress method unknown: " + uncompress)
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
	res, err := httpclient.Get(url, httpclient.Header{}, nil, user)
	if err != nil {
		//return nil, err
		return nil, errors.New("httpclient.Get returned: " + err.Error())
	}

	if res.StatusCode != 200 { //err in fetching data
		resbody, _ := ioutil.ReadAll(res.Body)
		return nil, errors.New(fmt.Sprintf("op=fetchFile, url=%s, res=%s", url, resbody))
	}

	return res.Body, err
}

// source:  http://stackoverflow.com/a/22259280
// TODO this is not shock related, need another package
func CopyFile(src, dst string) (int64, error) {
	src_file, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer src_file.Close()

	src_file_stat, err := src_file.Stat()
	if err != nil {
		return 0, err
	}

	if !src_file_stat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	dst_file, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dst_file.Close()
	return io.Copy(dst_file, src_file)
}
