package cwl

//"fmt"
//"strconv"

// YAML: null | Null | NULL | ~

type Null string

func (i *Null) IsCWLObject() {}

func (i *Null) GetClass() string      { return string(CWLNull) } // for CWLObject
func (i *Null) GetType() CWLType_Type { return CWLNull }
func (i *Null) String() string        { return "Null" }

func (i *Null) GetID() string  { return "" } // TODO deprecated functions
func (i *Null) SetID(x string) {}

func (i *Null) IsCWLMinimal() {}

func NewNull() (n *Null) {

	var null_nptr Null
	null_nptr = Null("Null")

	n = &null_nptr

	return
}

// https://mlafeldt.github.io/blog/decoding-yaml-in-go/

// No, use this: https://godoc.org/gopkg.in/yaml.v2#Marshaler
func (n *Null) MarshalYAML() (i interface{}, err error) {

	i = nil

	return
}

func (n *Null) MarshalJSON() (b []byte, err error) {

	b = []byte("null")
	return
}

func (n *Null) GetBSON() (result interface{}, err error) {
	result = nil
	return
}
