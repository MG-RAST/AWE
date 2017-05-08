package cwl

type CWL_object interface {
	CWL_minimal_interface
	GetClass() string
	GetId() string
	SetId(string)
	//is_Any()
}
