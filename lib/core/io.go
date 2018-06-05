package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/MG-RAST/golib/go-uuid/uuid"
	"net/url"
	"strings"
)

type IO struct {
	FileName      string                   `bson:"filename" json:"filename" mapstructure:"filename"`
	Name          string                   `bson:"name" json:"name" mapstructure:"name"`  // specifies abstract name of output as defined by the app
	AppPosition   int                      `bson:"appposition" json:"-" mapstructure:"-"` // specifies position in app output array
	Directory     string                   `bson:"directory" json:"directory" mapstructure:"directory"`
	Host          string                   `bson:"host" json:"host" mapstructure:"host"`
	Node          string                   `bson:"node" json:"node" mapstructure:"node"`
	Url           string                   `bson:"url"  json:"url" mapstructure:"url"` // can be shock or any other url
	Size          int64                    `bson:"size" json:"size" mapstructure:"size"`
	MD5           string                   `bson:"md5" json:"-" mapstructure:"-"`
	Cache         bool                     `bson:"cache" json:"cache" mapstructure:"cache"` // indicates that this files is "predata"" that needs to be cached
	Origin        string                   `bson:"origin" json:"origin" mapstructure:"origin"`
	Path          string                   `bson:"-" json:"-" mapstructure:"-"`
	Optional      bool                     `bson:"optional" json:"-" mapstructure:"-"`
	Nonzero       bool                     `bson:"nonzero"  json:"nonzero" mapstructure:"nonzero"`
	DataToken     string                   `bson:"datatoken"  json:"-" mapstructure:"-"`
	Intermediate  bool                     `bson:"Intermediate"  json:"-" mapstructure:"-"`
	Temporary     bool                     `bson:"temporary"  json:"temporary" mapstructure:"temporary"`
	ShockFilename string                   `bson:"shockfilename" json:"shockfilename" mapstructure:"shockfilename"`
	ShockIndex    string                   `bson:"shockindex" json:"shockindex" mapstructure:"shockindex"` // on input it indicates that Shock node has to be indexed by AWE server
	AttrFile      string                   `bson:"attrfile" json:"attrfile" mapstructure:"attrfile"`
	NoFile        bool                     `bson:"nofile" json:"nofile" mapstructure:"nofile"`
	Delete        bool                     `bson:"delete" json:"delete" mapstructure:"delete"` // speficies that this is a temorary node, to be deleted from shock on job completion
	Type          string                   `bson:"type" json:"type" mapstructure:"type"`
	NodeAttr      map[string]interface{}   `bson:"nodeattr" json:"nodeattr" mapstructure:"nodeattr"` // specifies attribute data to be stored in shock node (output only)
	FormOptions   map[string]string        `bson:"formoptions" json:"formoptions" mapstructure:"formoptions"`
	Uncompress    string                   `bson:"uncompress" json:"uncompress" mapstructure:"uncompress"` // tells AWE client to uncompress this file, e.g. "gzip"
	Indexes       map[string]shock.IdxInfo `bson:"-" json:"-" mapstructure:"-"`                            // copy of shock node.Indexes, not saved
}

type PartInfo struct {
	Input         string `bson:"input" json:"input" mapstructure:"input"`
	Index         string `bson:"index" json:"index" mapstructure:"index"`
	TotalIndex    int    `bson:"totalindex" json:"totalindex" mapstructure:"totalindex"`
	MaxPartSizeMB int    `bson:"maxpartsize_mb" json:"maxpartsize_mb" mapstructure:"maxpartsize_mb"`
	Options       string `bson:"options" json:"-" mapstructure:"-"`
}

// Deprecated JobDep struct uses deprecated TaskDep struct which uses the deprecated IOmap.  Maintained for backwards compatibility.
// Jobs that cannot be parsed into the Job struct, but can be parsed into the JobDep struct will be translated to the new Job struct.
// (=deprecated=)
type IOmap map[string]*IO // [filename]attributes

// (=deprecated=)
func NewIOmap() IOmap {
	return IOmap{}
}

// (=deprecated=)
func (i IOmap) Add(name string, host string, node string, md5 string, cache bool) {
	i[name] = &IO{FileName: name, Host: host, Node: node, MD5: md5, Cache: cache}
	return
}

// (=deprecated=)
func (i IOmap) Has(name string) bool {
	if _, has := i[name]; has {
		return true
	}
	return false
}

// (=deprecated=)
func (i IOmap) Find(name string) *IO {
	if val, has := i[name]; has {
		return val
	}
	return nil
}

func NewIO() *IO {
	return &IO{}
}

func (io *IO) Url2Shock() (err error) {
	if io.Url == "" {
		err = fmt.Errorf("(Url2Shock) url empty")
		return
	}
	u, _ := url.Parse(io.Url)
	if (u.Scheme == "") || (u.Host == "") || (u.Path == "") {
		err = fmt.Errorf("(Url2Shock) Not a valid url: %s", io.Url)
		return
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
	return
}

func (io *IO) DataUrl() (dataurl string, err error) {
	if io.Url != "" {
		// parse and test url
		err = io.Url2Shock()
		if err != nil {
			return
		}
		return io.Url, nil
	} else if (io.Host != "") && (io.Node != "") && (io.Node != "-") {
		io.Url = fmt.Sprintf("%s/node/%s%s", io.Host, io.Node, shock.DATA_SUFFIX)
		return io.Url, nil
	} else {
		// empty IO is valid
		return "", nil
	}
}

// this will update: io.Indexes, io.Size, io.MD5
func (io *IO) getShockNode() (node *shock.ShockNode, err error) {
	if io.Host == "" {
		err = errors.New("empty shock host")
		return
	}
	if (io.Node == "") || (io.Node == "-") {
		err = errors.New("empty node id")
		return
	}
	sc := shock.ShockClient{Host: io.Host, Token: io.DataToken}
	node, err = sc.Get_node(io.Node)
	if err != nil {
		return
	}
	// wait on file locked nodes
	if node.File.Locked != nil {
		node, err = sc.WaitFile(io.Node)
		if err != nil {
			return
		}
	}
	// cache node Indexes in-memory to prevent unnecessary lookup
	io.Indexes = node.Indexes
	return
}

func (io *IO) UpdateFileSize() (modified bool, err error) {
	modified = false
	if io.Size > 0 {
		return
	}
	var node *shock.ShockNode
	node, err = io.getShockNode() // this waits on locked file
	if err != nil {
		return
	}
	if (node.File.Size == 0) && node.File.CreatedOn.IsZero() {
		msg := fmt.Sprintf("node=%s: has no file", io.Node)
		if (node.Type == "parts") && (node.Parts != nil) {
			msg += fmt.Sprintf(", %d of %d parts completed", node.Parts.Length, node.Parts.Count)
		}
		err = errors.New(msg)
		return
	}
	// update, this needs to be saved to mongo
	io.Size = node.File.Size
	if md5, ok := node.File.Checksum["md5"]; ok {
		io.MD5 = md5
	}
	modified = true
	return
}

func (io *IO) IndexFile(indextype string) (idxInfo shock.IdxInfo, err error) {
	// see if already exists
	var hasIndex bool
	idxInfo, hasIndex = io.Indexes[indextype]
	if hasIndex && idxInfo.Locked == nil {
		return
	}

	// incomplete, update from shock
	var node *shock.ShockNode
	node, err = io.getShockNode() // this waits on locked file
	if err != nil {
		return
	}
	idxInfo, hasIndex = node.Indexes[indextype]

	// create and wait on index
	if !hasIndex || idxInfo.Locked != nil {
		sc := shock.ShockClient{Host: io.Host, Token: io.DataToken}
		// create missing index
		if !hasIndex {
			err = sc.ShockPutIndex(io.Node, indextype)
			if err != nil {
				return
			}
		}
		// wait on asynch indexing
		idxInfo, err = sc.WaitIndex(io.Node, indextype)
		if err != nil {
			return
		}
	}
	// bad state, unlocked but not complete
	if idxInfo.TotalUnits == 0 {
		err = fmt.Errorf("(IndexFile) index in bad state, TotalUnits is zero: node=%s, index=%s", io.Node, indextype)
		return
	}

	// update current io, this is in-memory only
	io.Indexes[indextype] = idxInfo
	return
}

func (io *IO) DeleteNode() (err error) {
	err = shock.ShockDelete(io.Host, io.Node, io.DataToken)
	return
}
