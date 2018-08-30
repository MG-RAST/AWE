package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InitialWorkDirRequirement
type InitialWorkDirRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
	Listing         interface{} `yaml:"listing,omitempty" bson:"listing,omitempty" json:"listing,omitempty" mapstructure:"listing,omitempty"` // TODO: array<File | Directory | Dirent | string | Expression> | string | Expression
}

func (c InitialWorkDirRequirement) GetId() string { return "" }

func NewInitialWorkDirRequirement(original interface{}) (r *InitialWorkDirRequirement, err error) {
	var requirement InitialWorkDirRequirement
	r = &requirement

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:

		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewInitialWorkDirRequirement) type error")
			return
		}

		listing, has_listing := original_map["listing"]
		if !has_listing {
			err = fmt.Errorf("(NewInitialWorkDirRequirement) Listing is missing")
			return
		}

		original_map["listing"], err = CreateListing(listing)
		if err != nil {
			err = fmt.Errorf("(NewInitialWorkDirRequirement) NewCWLType returned: %s", err.Error())
			return
		}

		err = mapstructure.Decode(original, &requirement)

		requirement.Class = "InitialWorkDirRequirement"

	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewInitialWorkDirRequirement) unknown type %s", reflect.TypeOf(original))
	}

	return
}

func NewListingFromInterface(original interface{}) (obj CWL_object, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	native_map, ok := original.(map[string]interface{})
	if ok {
		_, has_entry := native_map["entry"]
		if has_entry {
			obj, err = NewDirentFromInterface("", native_map)
			if err != nil {
				err = fmt.Errorf("(NewCWLType) NewDirent returned: %s", err.Error())
			}
			return
		}
	}
	var x CWLType
	x, err = NewCWLType("", original)
	if err != nil {
		err = fmt.Errorf("(NewListingFromInterface) NewCWLType returns: %s", err.Error())
		return
	}
	x_class := x.GetClass()
	switch x_class {
	case "File":
	case "Directory":
	case "Dirent":
	case "string":
	case "Expression":
	default:
		err = fmt.Errorf("(NewListingFromInterface) type %s is not a valid Listing (original_type: %s)", x_class, reflect.TypeOf(original))
		return
	}
	obj = x
	return

}

func CreateListing(original interface{}) (result interface{}, err error) {

	switch original.(type) {
	case []interface{}:
		array := []CWL_object{}

		original_array := original.([]interface{})

		for i, _ := range original_array {

			var new_listing CWL_object
			new_listing, err = NewListingFromInterface(original_array[i])
			if err != nil {
				err = fmt.Errorf("(CreateListingArray) NewListingFromInterface returns: %s", err.Error())
				return
			}
			array = append(array, new_listing)
		}
		result = array
	case interface{}:
		var new_listing CWL_object
		new_listing, err = NewListingFromInterface(original)
		if err != nil {
			err = fmt.Errorf("(CreateListingArray) NewListingFromInterface returns: %s", err.Error())
			return
		}
		result = new_listing
	default:
		err = fmt.Errorf("(CreateListingArray)")
	}

	return

}

// Listing: array<File | Directory | Dirent | string | Expression> | string | Expression
func (r *InitialWorkDirRequirement) Evaluate(inputs interface{}) (err error) {

	if inputs == nil {
		err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) no inputs")
		return
	}

	//fmt.Println("InitialWorkDirRequirement:")
	//spew.Dump(*r)

	if r.Listing == nil {
		return
	}

	listing := r.Listing
	switch listing.(type) {
	case []CWL_object:
		//fmt.Println("(InitialWorkDirRequirement/Evaluate) array")
		listing_array := listing.([]CWL_object)
		for i, _ := range listing_array {

			var element interface{}
			element = listing_array[i]
			spew.Dump(element)
			//fmt.Printf("(InitialWorkDirRequirement/Evaluate) type: %s\n", reflect.TypeOf(element))

			switch element.(type) {
			case *Dirent:
				// nothing to do
				return

				//fmt.Println("(InitialWorkDirRequirement/Evaluate) dirent")
				var element_dirent *Dirent
				element_dirent = element.(*Dirent)
				err = element_dirent.Evaluate(inputs)
				if err != nil {
					err = fmt.Errorf("(InitialWorkDirRequirement) element_dirent.Evaluate returned: %s", err.Error())
					return
				}
				//fmt.Println("(InitialWorkDirRequirement/Evaluate) element_dirent:")
				//spew.Dump(element_dirent)
				listing_array[i] = element_dirent
			case *File, *Directory:
				// nothing to do

			case *String:
				element_str := element.(*String)

				var original_expr *Expression
				original_expr = NewExpressionFromString(element_str.String())

				var new_value interface{}
				new_value, err = original_expr.EvaluateExpression(nil, inputs)
				if err != nil {
					err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) *String in listing_array EvaluateExpression returned: %s", err.Error())
					return
				}

				// verify return type:
				switch new_value.(type) {
				case *File:
					listing_array[i] = new_value.(*File)

				case *Directory:
					listing_array[i] = new_value.(*Directory)
				case *Dirent:
					listing_array[i] = new_value.(*Dirent)
				case string:
					listing_array[i] = NewString(new_value.(string))
				case *String:
					listing_array[i] = new_value.(*String)
				default:
					err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) (list mode) EvaluateExpression returned type %s, this is not expected", reflect.TypeOf(new_value))
					return

				}

			default:
				err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) type not supported: %s", reflect.TypeOf(element))
				return

			}
		} // end for
	case Dirent:
		// nothing to do
		return

		// listing_dirent := listing.(Dirent)
		// err = listing_dirent.Evaluate(inputs)
		// if err != nil {
		// 	err = fmt.Errorf("(InitialWorkDirRequirement) listing_dirent.Evaluate returned: %s", err.Error())
		// 	return
		// }
		// r.Listing = &listing_dirent
	case File, Directory:

		// nothing to do

	case String:

		listing_str := listing.(String)

		var original_expr *Expression
		original_expr = NewExpressionFromString(listing_str.String())

		var new_value interface{}
		new_value, err = original_expr.EvaluateExpression(nil, inputs)
		if err != nil {
			err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) String EvaluateExpression returned: %s", err.Error())
			return
		}

		// verify return type:
		switch new_value.(type) {
		case *File, *Directory, *Dirent, string, *String:
			// valid returns

		default:
			err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) (expression mode, String) EvaluateExpression returned type %s, this is not expected", reflect.TypeOf(listing))
			return

		}

		// set evaluated avlue
		r.Listing = new_value
	case *String:

		listing_str := listing.(*String)

		var original_expr *Expression
		original_expr = NewExpressionFromString(listing_str.String())

		var new_value interface{}
		new_value, err = original_expr.EvaluateExpression(nil, inputs)
		if err != nil {
			err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) *String EvaluateExpression returned: %s", err.Error())
			return
		}

		// verify return type:
		switch new_value.(type) {
		case *File, *Directory, *Dirent, string, *String, *Array:
			// valid returns

		default:
			err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) (expression mode, *String) EvaluateExpression returned type %s, this is not expected", reflect.TypeOf(new_value))
			return

		}

		// set evaluated avlue
		r.Listing = new_value

	default:

		err = fmt.Errorf("(InitialWorkDirRequirement/Evaluate) type not supported: %s", reflect.TypeOf(listing))
		return

	}
	//fmt.Println("(InitialWorkDirRequirement/Evaluate) end")
	return

}
