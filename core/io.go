package core

import (
	"fmt"
)

type IO struct {
	Name   string `bson:"name" json:"name"`
	Host   string `bson:"host" json:"host"`
	Node   string `bson:"node" json:"node"`
	Size   int64  `bson:"size" json:"size"`
	MD5    string `bson:"md5" json:"md5"`
	Cache  bool   `bson:"cache" json:"cache"`
	Origin string `bson:"origin" json:"origin"`
	Path   string `bson:"path" json:"-"`
}

type PartInfo struct {
	Input      string `bson:"input" json:"input"`
	Ouput      string `bson:"output" json:"output"`
	Index      string `bson:"index" json:"index"`
	TotalIndex int    `bson:"totalindex" json:"totalindex"`
	Options    string `bson:"options" json:"options"`
}

type IdxInfo struct {
	Type        string `bson: "index_type" json:"index_type"`
	TotalUnits  int    `bson: "total_units" json:"total_units"`
	AvgUnitSize int    `bson: "avg_unitsize" json:"avg_unitsize"`
}

type IOmap map[string]*IO

func NewIOmap() IOmap {
	return IOmap{}
}

func (i IOmap) Add(name string, host string, node string, params string, md5 string, cache bool) {
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

func (io *IO) Url() string {
	if io.Host != "" && io.Node != "" {
		return fmt.Sprintf("%s/node/%s?download", io.Host, io.Node)
	}
	return ""
}

func (io *IO) TotalUnits(indextype string) (count int, err error) {
	count, err = GetIndexUnits(indextype, io)
	return
}

func (io *IO) GetFileInfo() (size int64, idxinfo map[string]IdxInfo, err error) {
	var shocknode *ShockNode
	shocknode, err = GetShockNode(io.Host, io.Node)
	if err != nil {
		return
	}
	size = shocknode.File.Size
	idxinfo = shocknode.Indexes
	return
}
