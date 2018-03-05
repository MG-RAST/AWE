package cwl

//"github.com/davecgh/go-spew/spew"

// http://www.commonwl.org/v1.0/CommandLineTool.html#Dirent
type Dirent struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Entry        Expression `yaml:"entry" json:"entry" bson:"entry" mapstructure:"entry"`
	Entryname    Expression `yaml:"entryname" json:"entryname" bson:"entryname" mapstructure:"entryname"`
	Writable     bool       `yaml:"writable" json:"writable" bson:"writable" mapstructure:"writable"`
}
