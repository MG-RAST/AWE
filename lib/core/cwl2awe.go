package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/davecgh/go-spew/spew"
	//aw_sequences"os"
	requirements "github.com/MG-RAST/AWE/lib/core/cwl/requirements"
	"path"
	//"strconv"
	"strings"
)

type Helper struct {
	unprocessed_ws *map[string]*cwl.WorkflowStep
	processed_ws   *map[string]*cwl.WorkflowStep
	collection     *cwl.CWL_collection
	job            *Job
	AWE_tasks      *map[string]*Task
}

func parseSourceString(source string, id string) (linked_step_name string, fieldname string, err error) {

	if !strings.HasPrefix(source, "#") {
		err = fmt.Errorf("source has to start with # (%s)", id)
		return
	}

	source = strings.TrimPrefix(source, "#")

	source_array := strings.Split(source, "/")

	switch len(source_array) {
	case 0:
		err = fmt.Errorf("source empty (%s)", id)
	case 1:
		fieldname = source_array[0]
	case 2:
		linked_step_name = source_array[0]
		fieldname = source_array[1]
	default:
		err = fmt.Errorf("source has too many fields (%s)", id)
	}

	return
}

// func createAweTask(helper *Helper, cwl_tool *cwl.CommandLineTool, cwl_step *cwl.WorkflowStep, awe_task *Task) (err error) {
//
// 	// valueFrom is a StepInputExpressionRequirement
// 	// http://www.commonwl.org/v1.0/Workflow.html#StepInputExpressionRequirement
//
// 	collection := helper.collection
// 	//unprocessed_ws := helper.unprocessed_ws
// 	processed_ws := helper.processed_ws
//
// 	local_collection := cwl.NewCWL_collection()
// 	//workflowStepInputMap := make(map[string]cwl.WorkflowStepInput)
//
// 	linked_IO := make(map[string]*IO)
//
// 	spew.Dump(cwl_tool)
// 	spew.Dump(cwl_step)
//
// 	fmt.Println("=============================================== variables")
// 	// collect variables in local_collection
// 	for _, workflow_step_input := range cwl_step.In {
// 		// input is a cwl.WorkflowStepInput
//
// 		fmt.Println("workflow_step_input: ")
// 		spew.Dump(workflow_step_input)
// 		id := workflow_step_input.Id
//
// 		if id == "" {
// 			err = fmt.Errorf("(createAweTask) id is empty")
// 			return
// 		}
//
// 		fmt.Println("//////////////// step input: " + id)
//
// 		id_array := strings.Split(id, "/")
// 		id_base := id_array[len(id_array)-1]
//
// 		if len(workflow_step_input.Source) > 0 {
// 			logger.Debug(3, "len(workflow_step_input.Source): %d", len(workflow_step_input.Source))
// 			if len(workflow_step_input.Source) == 1 { // other only via MultipleInputFeature requirement
// 				for _, source := range workflow_step_input.Source { // #fieldname or #stepname.fieldname
// 					logger.Debug(3, "working on source: %s id=%s", source, id)
// 					var linked_step_name string
// 					var fieldname string
//
// 					var linked_step cwl.WorkflowStep
//
// 					// source: # is absolute path, without its is stepname/outputname
// 					if strings.HasPrefix(source, "#") {
//
// 						// check if this is a
// 						source_array := strings.Split(source, "/")
//
// 						switch len(source_array) {
// 						case 1:
// 							err = fmt.Errorf("len(source_array) == 1  with source_array[0]=%s", source_array[0])
// 							return
// 						case 2:
// 							fieldname = source_array[1]
//
// 							logger.Debug(3, "workflow input: %s", fieldname)
// 							// use workflow input as input
//
// 							var obj cwl.CWL_object
// 							job_input := *helper.collection.Job_input
// 							obj, ok := job_input[fieldname]
// 							if !ok {
//
// 								//for x, _ := range job_input {
// 								//	logger.Debug(3, "Job_input key: %s", x)
// 								//}
//
// 								logger.Debug(3, "(createAweTask) %s not found in Job_input", fieldname)
// 								continue
// 							}
//
// 							//obj_local := *obj
//
// 							logger.Debug(3, "rename %s -> %s", obj.GetId(), id_base)
//
// 							obj.SetId(id_base)
// 							fmt.Printf("(createAweTask) ADDEDb %s\n", id_base)
// 							fmt.Printf("(createAweTask) ADDED2b %s\n", obj.GetId())
// 							err = local_collection.Add(obj)
// 							if err != nil {
// 								return
// 							}
//
// 						case 3:
// 							linked_step_name = source_array[1]
// 							fieldname = source_array[2]
// 							logger.Debug(3, "workflowstep output: %s %s", source_array[1], source_array[2])
//
// 							this_linked_step, ok := (*processed_ws)[linked_step_name]
// 							if !ok {
//
// 								// recurse into depending step
// 								err = cwl_step_2_awe_task(helper, linked_step_name)
// 								if err != nil {
// 									err = fmt.Errorf("(createAweTask) source empty (%s) (%s)", id, err.Error())
// 									return
// 								}
//
// 								this2_linked_step, ok := (*processed_ws)[linked_step_name]
// 								if !ok {
// 									err = fmt.Errorf("(createAweTask) linked_step %s not found after processing", linked_step_name)
// 									return
// 								}
// 								linked_step = *this2_linked_step
// 							} else {
// 								linked_step = *this_linked_step
// 							}
// 							// linked_step is defined now.
// 							_ = linked_step
//
// 							// search output of linked_step
//
// 							linked_awe_task, ok := (*helper.AWE_tasks)[linked_step_name]
// 							if !ok {
// 								err = fmt.Errorf("(createAweTask) Could not find AWE task for linked step %s", linked_step_name)
// 								return
// 							}
// 							linked_task_id := linked_awe_task.Id
//
// 							linked_filename := ""
// 							for _, awe_output := range linked_awe_task.Outputs {
// 								// awe_output is an AWE IO{} object
// 								if awe_output.Name == fieldname {
// 									linked_filename = awe_output.FileName
// 									break
// 								}
// 							}
//
// 							if linked_filename == "" {
// 								err = fmt.Errorf("(createAweTask) Output %s not found", fieldname)
// 								return
// 							}
// 							//linked_output, err = linked_step.GetOutput(fieldname)
// 							//if err != nil {
// 							//	return
// 							//}
//
// 							// linked_output.Id provides
// 							linked_IO[id] = &IO{Origin: linked_task_id, FileName: linked_filename, Name: id}
//
// 						default:
// 							err = fmt.Errorf("source too long: %s", source)
// 							return
// 						}
//
// 						//source_object, err := collection.Get(source)
// 						//if err != nil {
// 						//	err = fmt.Errorf("Could not find source %s in collection", source)
//
// 						//}
//
// 					} else {
// 						err = fmt.Errorf("source without hash not implented yet: %s", source)
// 						return
// 					}
//
// 					//linked_step_name, fieldname, err = parseSourceString(source, id)
// 					//if err != nil {
// 					//	return
// 					//}
// 					//logger.Debug(3, "source: \"%s\" linked_step_name: \"%s\"  fieldname: \"%s\"", source, linked_step_name, fieldname)
//
// 					// find fieldname, may refer to other task, recurse in creation of that other task if necessary
//
// 				}
//
// 			} else {
// 				err = fmt.Errorf("MultipleInputFeatureRequirement not supported yet (id=%s)", id)
// 				return
// 			}
//
// 		} else {
//
// 			var obj *cwl.CWL_object
// 			obj, err = workflow_step_input.GetObject(helper.collection) // from ValueFrom or Default fields
// 			if err != nil {
// 				err = fmt.Errorf("(createAweTask) source for %s not found", id)
// 				return
// 			}
// 			obj_local := *obj
//
// 			//obj_int, ok := obj_local.(cwl.Int)
// 			//if ok {
// 			//	fmt.Printf("obj_int.Id: %s\n", obj_int.Id)
// 			//	obj_int.Id = "test"
// 			//	fmt.Printf("obj_int.Id: %s\n", obj_int.Id)
// 			//	fmt.Printf("obj_int.Value: %d\n", obj_int.Value)
// 			//	spew.Dump(obj_int)
// 			//	obj_int.SetId(id)
// 			//	fmt.Printf("obj_int]]]]]]] %s\n", obj_int.GetId())
// 			//}
//
// 			spew.Dump(obj_local)
// 			obj_local.SetId(id)
// 			spew.Dump(obj_local)
// 			fmt.Printf("ADDED %s\n", id)
// 			fmt.Printf("ADDED2 %s\n", obj_local.GetId())
// 			err = local_collection.Add(obj_local) // add object with local input name
// 			if err != nil {
// 				return
// 			}
// 		}
//
// 		//workflowStepInputMap[id] = input
// 		// use content of input instead ": local_collection.Add(input)
// 		//fmt.Println("ADD local_collection.WorkflowStepInputs")
// 		//spew.Dump(local_collection.WorkflowStepInputs)
// 	}
// 	//fmt.Println("workflowStepInputMap:")
// 	//spew.Dump(workflowStepInputMap)
//
// 	// copy cwl_tool to AWE via step
// 	command_line := ""
// 	fmt.Println("=============================================== expected inputs")
// 	for _, expected_input := range cwl_tool.Inputs {
// 		// input is a cwl.CommandInputParameter
// 		fmt.Println("expected_input:")
// 		spew.Dump(expected_input)
//
// 		id := expected_input.Id
// 		_ = id
// 		expected_types := expected_input.Type
// 		//if len(expected_input.Type) > 1 {
// 		//	err = fmt.Errorf("Not yet supported: len(expected_input.Type) > 1")
// 		//	return
// 		//}
//
// 		//expected_input_parameter_type_0 := expected_input.Type[0]
// 		//_ = expected_input_parameter_type_0
// 		//input_optional := strings.HasSuffix(expected_input_type_0, "?") TODO
// 		//expected_input_type := strings.ToLower(strings.TrimSuffix(expected_input_type_0, "?"))TODO
// 		//_ = input_optional
// 		//_ = expected_input_type
//
// 		// TODO lookup id in workflow step input
//
// 		//fmt.Println("local_collection.WorkflowStepInputs")
// 		//spew.Dump(local_collection.WorkflowStepInputs)
// 		//workflow_step_input, ok := workflowStepInputMap[id]
// 		//actual_input, xerr := local_collection.Get(id)
// 		//if xerr != nil {
// 		//	err = fmt.Errorf("%s not found in workflowStepInputMap", id)
// 		//	return
// 		//}
// 		//fmt.Println("workflow_step_input: ")
// 		//spew.Dump(workflow_step_input)
//
// 		id_array := strings.Split(id, "/")
// 		id_base := id_array[len(id_array)-1]
//
// 		input_string := ""
// 		found_input := false
// 		var obj *cwl.CWL_object
//
// 		logger.Debug(3, "try to find %s", id_base)
// 		for key, _ := range local_collection.All {
// 			logger.Debug(3, "local_collection key %s", key)
// 		}
//
// 		obj, err = local_collection.Get(id_base)
//
// 		if err == nil {
// 			found_input = true
// 			logger.Debug(3, "FOUND key %s in local_collection", id_base)
// 		} else {
// 			logger.Debug(3, "DID NOT FIND key %s", id_base)
// 		}
//
// 		if !found_input {
// 			// try linked_IO
//
// 			// TODO compare type
// 			io_struct, ok := linked_IO[id]
// 			if ok {
// 				awe_task.Inputs = append(awe_task.Inputs, io_struct)
// 				// TODO
// 				// io_struct.DataToken = ..
// 				found_input = true
// 				logger.Debug(3, "FOUND found_input %s in linked_IO", id)
// 			}
// 		}
//
// 		if !found_input {
//
// 			this_job_input, ok := (*collection.Job_input)[id_base]
// 			if ok {
//
// 				obj_nptr := (this_job_input).(cwl.CWL_object)
//
// 				obj = &obj_nptr
// 				found_input = true
// 				logger.Debug(3, "FOUND found_input %s in Job_input", id_base)
// 			} else {
// 				fmt.Println("was searching for: " + id_base)
// 				fmt.Println("collection.Job_input:")
// 				spew.Dump(collection.Job_input)
// 			}
//
// 		}
//
// 		if !found_input {
// 			// try the default value
// 			default_input := expected_input.Default
// 			if default_input != nil {
//
// 				// probably not a file, but some argument
// 				logger.Debug(3, "expected_input.Default found something")
// 				obj_nptr := (*default_input).(cwl.CWL_object)
// 				obj = &obj_nptr
// 				found_input = true
// 				logger.Debug(3, "FOUND found_input %s in expected_input.Default", id_base)
// 			} else {
// 				logger.Debug(3, "expected_input.Default found nothing")
// 			}
// 		} else {
// 			logger.Debug(3, "skipping expected_input.Default")
// 		}
//
// 		if !found_input {
// 			// check if argument is optional
// 			if !cwl.HasCommandInputParameterType(&expected_types, cwl.CWL_null) {
// 				err = fmt.Errorf("%s not found in local_collection or linked_IO, and no default found", id)
//
// 				fmt.Println("-------------------------------------------------")
// 				spew.Dump(local_collection)
// 				spew.Dump(linked_IO)
// 				return
// 			}
// 			// this argument is optional, continue....
// 			continue
//
// 		}
//
// 		obj_class := (*obj).GetClass()
// 		switch obj_class {
//
// 		case cwl.CWL_File:
//
// 			var file_obj *cwl.File
// 			file_obj, ok := (*obj).(*cwl.File) //local_collection.GetFile(id_base)
// 			if !ok {
// 				err = fmt.Errorf("File %s not found in local_collection ", id_base)
// 				return
// 			}
//
// 			// TODO HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE HERE
// 			input_string = file_obj.Basename
//
// 			awe_input := NewIO()
// 			awe_input.FileName = file_obj.Basename
// 			awe_input.Name = file_obj.Id
// 			awe_input.Host = file_obj.Host
// 			awe_input.Node = file_obj.Node
// 			//awe_input.Url=input_file.
// 			awe_input.DataToken = file_obj.Token
//
// 			awe_task.Inputs = append(awe_task.Inputs, awe_input)
//
// 			if input_string == "" {
// 				err = fmt.Errorf("input_string File is empty")
// 				return
// 			}
//
// 		case cwl.CWL_string:
//
// 			var string_obj *cwl.String
// 			string_obj, ok := (*obj).(*cwl.String) //local_collection.GetString(id_base)
// 			if !ok {
// 				err = fmt.Errorf("String %s not found in local_collection ", id_base)
// 				return
// 			}
// 			input_string = string_obj.String()
//
// 			if input_string == "" {
// 				err = fmt.Errorf("input_string String is empty")
// 				return
// 			}
//
// 		case cwl.CWL_int:
//
// 			var int_obj *cwl.Int
// 			int_obj, ok := (*obj).(*cwl.Int) //local_collection.GetInt(id_base)
// 			if !ok {
// 				err = fmt.Errorf("Int %s not found in local_collection ", id_base)
// 				return
// 			}
// 			input_string = int_obj.String()
//
// 			if input_string == "" {
// 				err = fmt.Errorf("input_string Int is empty")
// 				return
// 			}
//
// 		case cwl.CWL_boolean:
// 			input_string = ""
//
// 		default:
//
// 			err = fmt.Errorf("(expected inputs) not implemented yet (%s) id=%s", obj_class, id_base)
// 			return
// 		}
//
// 		input_binding := expected_input.InputBinding
//
// 		//if input_binding != nil {
// 		prefix := input_binding.Prefix
// 		if prefix != "" {
//
// 			if command_line != "" {
// 				command_line = command_line + " "
// 			}
// 			command_line = command_line + prefix
//
// 			if input_string != "" {
// 				command_line = command_line + " " + input_string
// 			}
// 		}
//
// 		//}
// 		// Id:  #fieldname or #stepname.fieldname
// 		//switch input.Source {
// 		//case
//
// 		//}
// 		fmt.Println("command_line: ", command_line)
// 	}
//
// 	awe_task.Cmd.Name = cwl_tool.BaseCommand[0]
// 	if len(cwl_tool.BaseCommand) > 1 {
// 		awe_task.Cmd.Args = strings.Join(cwl_tool.BaseCommand[1:], " ")
// 	}
// 	awe_task.Cmd.Args += local_collection.Evaluate(command_line)
//
// 	for _, requirement := range cwl_tool.Hints {
// 		class := requirement.GetClass()
// 		switch class {
// 		case "DockerRequirement":
//
// 			//req_nptr := *requirement
//
// 			dr := requirement.(*requirements.DockerRequirement)
// 			//if !ok {
// 			//	err = fmt.Errorf("Could not cast into DockerRequirement")
// 			//	return
// 			//}
// 			if dr.DockerPull != "" {
// 				awe_task.Cmd.DockerPull = dr.DockerPull
// 			}
//
// 			if dr.DockerLoad != "" {
// 				err = fmt.Errorf("DockerRequirement.DockerLoad not supported yet")
// 				return
// 			}
// 			if dr.DockerFile != "" {
// 				err = fmt.Errorf("DockerRequirement.DockerFile not supported yet")
// 				return
// 			}
// 			if dr.DockerImport != "" {
// 				err = fmt.Errorf("DockerRequirement.DockerImport not supported yet")
// 				return
// 			}
// 			if dr.DockerImageId != "" {
// 				err = fmt.Errorf("DockerRequirement.DockerImageId not supported yet")
// 				return
// 			}
// 		default:
// 			err = fmt.Errorf("Requirement \"%s\" not supported", class)
// 			return
// 		}
// 	}
//
// 	// collect tool outputs
// 	tool_outputs := make(map[string]*cwl.CommandOutputParameter)
// 	for _, output := range cwl_tool.Outputs {
// 		tool_outputs[output.Id] = &output
// 	}
//
// 	process := cwl_step.Run
// 	process_class := (*process).GetClass()
//
// 	process_prefix := ""
//
// 	switch process_class {
// 	case "ProcessPointer":
// 		process_nptr := *process
// 		pp := process_nptr.(*cwl.ProcessPointer)
//
// 		process_prefix = pp.Value
//
// 	default:
// 		err = fmt.Errorf("Process type %s not supported yet", process_class)
// 		return
// 	}
//
// 	for _, output := range cwl_step.Out {
// 		// output is a WorkflowStepOutput, example: "#main/qc/assembly" convert to run_id + basename(output.Id)
// 		output_id := output.Id
//
// 		output_id_array := strings.Split(output_id, "/")
// 		output_id_base := output_id_array[len(output_id_array)-1]
//
// 		output_source_id := process_prefix + "/" + output_id_base
//
// 		// lookup glob of CommandLine tool
// 		cmd_out_param, ok := tool_outputs[output_source_id]
// 		if !ok {
//
// 			for key, _ := range tool_outputs {
// 				logger.Debug(3, "tool_outputs key: %s", key)
// 			}
//
// 			err = fmt.Errorf("output_id %s not found in tool_outputs", output_id)
// 			return
// 		}
//
// 		awe_output := NewIO()
// 		awe_output.Name = output_id
//
// 		// need to switch, but I do no know what type to expect. CommandOutputBinding.Type is []CommandOutputParameterType
// 		//switch cmd_out_param
//
// 		glob_array := cmd_out_param.OutputBinding.Glob
// 		switch len(glob_array) {
// 		case 0:
// 			err = fmt.Errorf("output_id %s glob empty", output_id)
// 			return
// 		case 1:
// 			glob := glob_array[0]
//
// 			fmt.Println("READ local_collection.WorkflowStepInputs")
// 			spew.Dump(local_collection.WorkflowStepInputs)
//
// 			glob_evaluated := local_collection.Evaluate(glob.String())
// 			fmt.Printf("glob: %s -> %s", glob.String(), glob_evaluated)
// 			// TODO evalue glob from workflowStepInputMap
// 			//step_input, xerr := local_collection.Get(id)
// 			//if xerr != nil {
// 			//	err = xerr
// 			//	return
// 			//}
// 			//blubb := step_input.Default.String()
//
// 			awe_output.FileName = glob_evaluated
// 		default:
// 			err = fmt.Errorf("output_id %s too many globs, not supported yet", output_id)
// 			return
// 		}
//
// 		if awe_output.Host == "" {
// 			awe_output.Host = helper.job.ShockHost
// 		}
// 		awe_task.Outputs = append(awe_task.Outputs, awe_output)
// 	}
//
// 	//fmt.Println("=============================================== expected inputs")
//
// 	return
// }
//
// func CommandLineTool2awe_task(helper *Helper, step *cwl.WorkflowStep, cmdlinetool *cwl.CommandLineTool) (awe_task *Task, err error) {
//
// 	//processed_ws := helper.processed_ws
// 	//collection := helper.collection
// 	job := helper.job
//
// 	// TODO detect identifier URI (http://www.commonwl.org/v1.0/SchemaSalad.html#Identifier_resolution)
// 	// TODO: strings.Contains(step.Run, ":") or use split
//
// 	if len(cmdlinetool.BaseCommand) == 0 {
// 		err = errors.New("(cwl_step_2_awe_task) Run string empty")
// 		return
// 	}
//
// 	pos := len(*helper.processed_ws)
// 	logger.Debug(1, "pos: %d", pos)
//
// 	pos_str := strconv.Itoa(pos)
// 	logger.Debug(1, "pos_str: %s", pos_str)
//
// 	awe_task, err = NewTask(job, pos_str)
// 	if err != nil {
// 		err = fmt.Errorf("(cwl_step_2_awe_task) Task creation failed: %v", err)
// 		return
// 	}
// 	awe_task.Init(job)
// 	logger.Debug(1, "(cwl_step_2_awe_task) Task created: %s", awe_task.Id)
//
// 	(*helper.AWE_tasks)[job.Id] = awe_task
// 	awe_task.JobId = job.Id
//
// 	err = createAweTask(helper, cmdlinetool, step, awe_task)
// 	if err != nil {
// 		return
// 	}
//
// 	return
// }

// func cwl_step_2_awe_task(helper *Helper, step_id string) (err error) {
// 	logger.Debug(1, "(cwl_step_2_awe_task) step_id: "+step_id)
//
// 	processed_ws := helper.processed_ws
// 	collection := helper.collection
// 	job := helper.job
//
// 	step, ok := (*helper.unprocessed_ws)[step_id]
// 	if !ok {
// 		err = fmt.Errorf("Step %s not found in unprocessed_ws", step_id)
// 		return
// 	}
// 	delete(*helper.unprocessed_ws, step_id) // will be added at the end of this function to processed_ws
//
// 	spew.Dump(step)
// 	process_ptr := step.Run
// 	process := *process_ptr
// 	//fmt.Println("run: " + step.Run)
//
// 	process_type := process.GetClass()
//
// 	switch process_type {
// 	case "CommandLineTool":
//
// 		cmdlinetool, ok := process.(*cwl.CommandLineTool)
// 		if !ok {
// 			err = fmt.Errorf("process type is not a CommandLineTool: %s", err.Error())
// 			return
// 		}
//
// 		awe_task, xerr := CommandLineTool2awe_task(helper, step, cmdlinetool)
// 		if xerr != nil {
// 			err = xerr
// 			return
// 		}
// 		job.Tasks = append(job.Tasks, awe_task)
//
// 		(*processed_ws)[step.Id] = step
// 		logger.Debug(1, "(cwl_step_2_awe_task) LEAVING , step_id: "+step_id)
// 	case "ProcessPointer":
//
// 		pp, ok := process.(*cwl.ProcessPointer)
// 		if !ok {
// 			err = fmt.Errorf("ProcessPointer error")
// 			return
// 		}
//
// 		pp_value := pp.Value
//
// 		the_process_ptr, xerr := collection.Get(pp_value)
// 		if xerr != nil {
// 			err = xerr
// 			return
// 		}
// 		the_process := *the_process_ptr
//
// 		the_process_class := the_process.GetClass()
//
// 		// CommandLineTool | ExpressionTool | Workflow
// 		switch the_process_class {
// 		case "CommandLineTool":
// 			logger.Debug(1, "(cwl_step_2_awe_task) got CommandLineTool")
//
// 			cmdlinetool, ok := the_process.(*cwl.CommandLineTool)
//
// 			if !ok {
// 				err = fmt.Errorf("(cwl_step_2_awe_task) casting error")
// 				return
// 			}
//
// 			awe_task, xerr := CommandLineTool2awe_task(helper, step, cmdlinetool)
// 			if xerr != nil {
// 				err = xerr
// 				return
// 			}
// 			job.Tasks = append(job.Tasks, awe_task)
//
// 			(*processed_ws)[step.Id] = step
//
// 			//case cwl.ExpressionTool:
// 			//logger.Debug(1, "(cwl_step_2_awe_task) got ExpressionTool")
// 		case "Workflow":
// 			logger.Debug(1, "(cwl_step_2_awe_task) got Workflow")
// 		default:
// 			err = fmt.Errorf("ProcessPointer error")
// 			return
// 		}
//
// 	default:
// 		err = fmt.Errorf("process type %s unknown", process_type)
// 		return
//
// 	}
//
// 	//if strings.HasPrefix(step.Run, "#") {
// 	// '#' it is a URI relative fragment identifier
// 	//	cmdlinetool_str = strings.TrimPrefix(step.Run, "#")
//
// 	//}
//
// 	return
// }

func CWL_input_check(cwl_workflow *cwl.Workflow, collection *cwl.CWL_collection) (err error) {

	for _, input := range cwl_workflow.Inputs {
		// input is a cwl.InputParameter object

		spew.Dump(input)

		id := input.Id
		logger.Debug(3, "Parsing workflow input %s", id)
		//if strings.HasPrefix(id, "#") {
		//	workflow_prefix := cwl_workflow.Id + "/"
		//	if strings.HasPrefix(id, workflow_prefix) {
		//		id = strings.TrimPrefix(id, workflow_prefix)
		//		logger.Debug(3, "using new id : %s", id)
		//	} else {
		//		err = fmt.Errorf("prefix of id %s does not match workflow id %s with workflow_prefix: %s", id, cwl_workflow.Id, workflow_prefix)
		//		return
		//	}
		//}

		id_base := path.Base(id)
		expected_types := input.Type

		job_input := *(collection.Job_input)
		obj_ref, ok := job_input[id_base]
		if !ok {

			if cwl.HasInputParameterType(expected_types, cwl_types.CWL_null) { // null type means parameter is optional
				continue
			}

			fmt.Printf("-------Collection")
			spew.Dump(collection.All)

			fmt.Printf("-------")
			err = fmt.Errorf("value for workflow input \"%s\" not found", id)
			return

		}

		//obj := *obj_ref
		obj_type := obj_ref.GetClass()
		logger.Debug(1, "obj_type: %s", obj_type)
		found_type := cwl.HasInputParameterType(expected_types, obj_type)

		if !found_type {
			//if strings.ToLower(obj_type) != strings.ToLower(expected_types) {
			fmt.Printf("object found: ")
			spew.Dump(obj_ref)
			expected_types_str := ""

			for _, elem := range *expected_types {
				expected_types_str += "," + elem.Type
			}
			fmt.Printf("cwl_workflow.Inputs")
			spew.Dump(cwl_workflow.Inputs)
			err = fmt.Errorf("Input %s expects types %s, but got %s)", id, expected_types_str, obj_type)
			return
		}

	}
	return
}

func CWL2AWE(_user *user.User, files FormFiles, cwl_workflow *cwl.Workflow, collection *cwl.CWL_collection) (job *Job, err error) {

	//CommandLineTools := collection.CommandLineTools

	// check that all expected workflow inputs exist and that they have the correct type
	logger.Debug(1, "CWL2AWE starting")

	err = CWL_input_check(cwl_workflow, collection)
	if err != nil {
		return
	}

	//os.Exit(0)
	job = NewJob()
	job.CWL_workflow = cwl_workflow
	logger.Debug(1, "Job created")

	found_ShockRequirement := false
	for _, r := range cwl_workflow.Requirements { // TODO put ShockRequirement in Hints
		req, ok := r.(cwl.Requirement)
		if !ok {
			err = fmt.Errorf("not a requirement")
			return
		}
		switch req.GetClass() {
		case "ShockRequirement":
			sr, ok := req.(requirements.ShockRequirement)
			if !ok {
				err = fmt.Errorf("Could not assert ShockRequirement")
				return
			}

			job.ShockHost = sr.Host
			found_ShockRequirement = true

		}
	}

	if !found_ShockRequirement {
		//err = fmt.Errorf("ShockRequirement has to be provided in the workflow object")
		//return
		job.ShockHost = "http://shock:7445"

	}
	logger.Debug(1, "Requirements checked")

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(_user.Uuid)
	job.Acl.Set(_user.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	// TODO first check that all resources are available: local files and remote links

	helper := Helper{}

	processed_ws := make(map[string]*cwl.WorkflowStep)
	unprocessed_ws := make(map[string]*cwl.WorkflowStep)
	awe_tasks := make(map[string]*Task)
	helper.processed_ws = &processed_ws
	helper.unprocessed_ws = &unprocessed_ws
	helper.collection = collection
	helper.job = job
	helper.AWE_tasks = &awe_tasks

	for _, step := range cwl_workflow.Steps {
		awe_task := NewTask(job, step.Id)
		awe_task.workflowStep = &step
		job.Tasks = append(job.Tasks, awe_task)
	}

	_, err = job.Init()
	if err != nil {
		err = fmt.Errorf("job.Init() failed: %v", err)
		return
	}
	logger.Debug(1, "Init called")

	err = job.Mkdir()
	if err != nil {
		err = errors.New("(CWL2AWE) error creating job directory, error=" + err.Error())
		return
	}

	err = job.UpdateFile(files, "cwl") // TODO that may not make sense. Check if I should store AWE job.
	if err != nil {
		err = errors.New("error in UpdateFile, error=" + err.Error())
		return
	}

	spew.Dump(job)

	logger.Debug(1, "job.Id: %s", job.Id)
	err = job.Save()
	if err != nil {
		err = errors.New("error in job.Save(), error=" + err.Error())
		return
	}
	return

}
