package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/MG-RAST/golib/go-uuid/uuid"
	"net/url"
	"strings"
)

type IO struct {
	FileName      string                 `bson:"filename" json:"filename"`
	Name          string                 `bson:"name" json:"name"`     // specifies abstract name of output as defined by the app
	AppPosition   int                    `bson:"appposition" json:"-"` // specifies position in app output array
	Directory     string                 `bson:"directory" json:"directory"`
	Host          string                 `bson:"host" json:"host"`
	Node          string                 `bson:"node" json:"node"`
	Url           string                 `bson:"url"  json:"url"` // can be shock or any other url
	Size          int64                  `bson:"size" json:"size"`
	MD5           string                 `bson:"md5" json:"-"`
	Cache         bool                   `bson:"cache" json:"cache"` // indicates that this files is "predata"" that needs to be cached
	Origin        string                 `bson:"origin" json:"origin"`
	Path          string                 `bson:"path" json:"-"`
	Optional      bool                   `bson:"optional" json:"-"`
	Nonzero       bool                   `bson:"nonzero"  json:"nonzero"`
	DataToken     string                 `bson:"datatoken"  json:"-"`
	Intermediate  bool                   `bson:"Intermediate"  json:"-"`
	Temporary     bool                   `bson:"temporary"  json:"temporary"`
	ShockFilename string                 `bson:"shockfilename" json:"shockfilename"`
	ShockIndex    string                 `bson:"shockindex" json:"shockindex"` // on input it indicates that Shock node has to be indexed by AWE server
	AttrFile      string                 `bson:"attrfile" json:"attrfile"`
	NoFile        bool                   `bson:"nofile" json:"nofile"`
	Delete        bool                   `bson:"delete" json:"delete"`
	Type          string                 `bson:"type" json:"type"`
	NodeAttr      map[string]interface{} `bson:"nodeattr" json:"nodeattr"` // specifies attribute data to be stored in shock node (output only)
	FormOptions   map[string]string      `bson:"formoptions" json:"formoptions"`
	Uncompress    string                 `bson:"uncompress" json:"uncompress"` // tells AWE client to uncompress this file, e.g. "gzip"
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
	i[name] = &IO{FileName: name, Host: host, Node: node, MD5: md5, Cache: cache}
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

func (io *IO) DataUrl() (dataurl string, err error) {
	if io.Url != "" {
		// parse and test url
		u, _ := url.Parse(io.Url)
		if (u.Scheme == "") || (u.Host == "") || (u.Path == "") {
			return "", errors.New("Not a valid url: " + io.Url)
		}
		// get shock info from url
		if (io.Host == "") || (io.Node == "") || (io.Node == "-") {
			trimPath := strings.Trim(u.Path, "/")
			cleanUuid := strings.Trim(strings.TrimPrefix(trimPath, "node"), "/")
			// appears to be a shock url
			if (cleanUuid != trimPath) && (uuid.Parse(cleanUuid) != nil) {
				io.Host = u.Scheme + "://" + u.Host
				io.Node = cleanUuid
			}
		}
		return io.Url, nil
	} else if (io.Host != "") && (io.Node != "") && (io.Node != "-") {
		io.Url = fmt.Sprintf("%s/node/%s?download", io.Host, io.Node)
		return io.Url, nil
	} else {
		// empty IO is valid
		return "", nil
	}
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
		logger.Error(fmt.Sprintf("GetFileSize error: %s, node: %s", err.Error(), io.Node))
		return -1
	}
	io.Size = shocknode.File.Size
	return io.Size
}

func (io *IO) GetIndexInfo() (idxinfo map[string]shock.IdxInfo, err error) {
	var shocknode *shock.ShockNode
	shocknode, err = io.GetShockNode()
	if err != nil {
		return
	}
	idxinfo = shocknode.Indexes
	return
}

func (io *IO) GetShockNode() (node *shock.ShockNode, err error) {
	if io.Host == "" {
		return nil, errors.New("empty shock host")
	}
	if io.Node == "-" {
		return nil, errors.New("empty node id")
	}
	return shock.ShockGet(io.Host, io.Node, io.DataToken)
}

func (io *IO) GetIndexUnits(indextype string) (totalunits int, err error) {
	var shocknode *shock.ShockNode
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

func (io *IO) DeleteNode() (nodeid string, err error) {
	if io.Delete {
		if err := shock.ShockDelete(io.Host, io.Node, io.DataToken); err != nil {
			return io.Node, err
		}
		return io.Node, nil
	}
	return "", nil
}
