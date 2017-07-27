package cwl

type Directory struct {
	Id       string         `yaml:"id"`
	Location string         `yaml:"location"`
	Path     string         `yaml:"path"`
	Basename string         `yaml:"basename"`
	Listing  []CWL_location `yaml:"basename"`
}

func (d Directory) GetClass() string    { return "Directory" }
func (d Directory) GetId() string       { return d.Id }
func (d Directory) String() string      { return d.Path }
func (d Directory) GetLocation() string { return d.Location } // for CWL_location
