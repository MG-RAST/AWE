package core

import (
	"errors"
	"fmt"
	"strings"
)

type IO struct {
	Name      string `bson:"name" json:"-"`
	Host      string `bson:"host" json:"host"`
	Node      string `bson:"node" json:"node"`
	Url       string `bson:"url"  json:"url"`
	Size      int64  `bson:"size" json:"size"`
	MD5       string `bson:"md5" json:"-"`
	Cache     bool   `bson:"cache" json:"-"`
	Origin    string `bson:"origin" json:"origin"`
	Path      string `bson:"path" json:"-"`
	Optional  bool   `bson:"optional" json:"-"`
	Nonzero   bool   `bson:"nonzero"  json:"nonzero"`
	DataToken string `bson:"datatoken"  json:"-"`
}

type PartInfo struct {
	Input         string `bson:"input" json:"input"`
	Index         string `bson:"index" json:"index"`
	TotalIndex    int    `bson:"totalindex" json:"totalindex"`
	MaxPartSizeMB int    `bson:"maxpartsize_mb" json:"maxpartsize_mb"`
	Options       string `bson:"options" json:"-"`
}

type IOmap map[string]*IO // [filename]attributes

func NewIOmap() IOmap {
	return IOmap{}
}

func (i IOmap) Add(name string, host string, node string, md5 string, cache bool) {
	i[name] = &IO{Name: name, Host: host, Node: node, MD5: md5, Cache: cache}
	return
}

func (i IOmap) Has(name string) bool {
	if _, has := i[name]; has {
		return true
	}
	return false
}

func (i IOmap) Find(name string) *IO {
	if val, has := i[name]; has {
		return val
	}
	return nil
}

func NewIO() *IO {
	return &IO{}
}

func (io *IO) DataUrl() string {
	if io.Url != "" {
		if io.Host == "" || io.Node == "-" {
			parts := strings.Split(io.Url, "/")
			io.Host = "http://" + parts[2]
			io.Node = strings.Split(parts[4], "?")[0]
		}
		return io.Url
	} else {
		if io.Host != "" && io.Node != "-" {
			downloadUrl := fmt.Sprintf("%s/node/%s?download", io.Host, io.Node)
			io.Url = downloadUrl
			return downloadUrl
		}
	}
	return ""
}

func (io *IO) TotalUnits(indextype string) (count int, err error) {
	count, err = io.GetIndexUnits(indextype)
	return
}

func (io *IO) GetFileSize() int64 {
	if io.Size > 0 {
		return io.Size
	}
	shocknode, err := io.GetShockNode()
	if err != nil {
		return -1
	}
	io.Size = shocknode.File.Size
	return io.Size
}

func (io *IO) GetIndexInfo() (idxinfo map[string]IdxInfo, err error) {
	var shocknode *ShockNode
	shocknode, err = io.GetShockNode()
	if err != nil {
		return
	}
	idxinfo = shocknode.Indexes
	return
}

func (io *IO) GetShockNode() (node *ShockNode, err error) {
	if io.Host == "" || io.Node == "" {
		return nil, errors.New("empty shock host or node id")
	}
	return ShockGet(io.Host, io.Node, io.DataToken)
}

func (io *IO) GetIndexUnits(indextype string) (totalunits int, err error) {
	var shocknode *ShockNode
	shocknode, err = io.GetShockNode()
	if err != nil {
		return
	}
	if _, ok := shocknode.Indexes[indextype]; ok {
		if shocknode.Indexes[indextype].TotalUnits > 0 {
			return int(shocknode.Indexes[indextype].TotalUnits), nil
		}
	}
	return 0, errors.New("invalid totalunits for shock node:" + io.Node)
}
