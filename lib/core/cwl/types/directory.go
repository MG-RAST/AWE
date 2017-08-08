package cwl

type Directory struct {
	Id       string         `yaml:"id,omitempty" json:"id,omitempty" bson:"id,omitempty"`
	Location string         `yaml:"location,omitempty" json:"location,omitempty" bson:"location,omitempty"`
	Path     string         `yaml:"path,omitempty" json:"path,omitempty" bson:"path,omitempty"`
	Basename string         `yaml:"basename,omitempty" json:"basename,omitempty" bson:"basename,omitempty"`
	Listing  []CWL_location `yaml:"listing,omitempty" json:"listing,omitempty" bson:"listing,omitempty"`
}

func (d Directory) GetClass() string    { return "Directory" }
func (d Directory) GetId() string       { return d.Id }
func (d Directory) String() string      { return d.Path }
func (d Directory) GetLocation() string { return d.Location } // for CWL_location
