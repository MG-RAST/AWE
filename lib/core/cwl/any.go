package cwl

//type Any interface {
//	CWL_object
//	String() string
//}

type Any interface {
	CWL_minimal_interface
}

func NewAny(native interface{}) (any Any, err error) {

	cwl_type, err := NewCWLType(native)
	if err == nil {
		any = cwl_type
		return
	}

	// TODO File, Directory

	return

}
