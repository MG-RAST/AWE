package cwl

type CWL_collection struct {
	Workflows        map[string]Workflow
	CommandLineTools map[string]CommandLineTool
	Files            map[string]File
	All              map[string]*CWL_object
}

func (c CWL_collection) add(obj CWL_object) {

	id := obj.GetId()
	switch obj.GetClass() {
	case "Workflow":
		c.Workflows[id] = obj.(Workflow)
	case "CommandLineTool":
		c.CommandLineTools[id] = obj.(CommandLineTool)
	case "File":
		c.Files[id] = obj.(File)
	}
	c.All[obj.GetId()] = &obj
}

func NewCWL_collection() (collection CWL_collection) {
	collection = CWL_collection{}

	collection.Workflows = make(map[string]Workflow)
	collection.CommandLineTools = make(map[string]CommandLineTool)
	collection.Files = make(map[string]File)
	collection.All = make(map[string]*CWL_object)
	return
}
