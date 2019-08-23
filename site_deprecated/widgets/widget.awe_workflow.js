(function () {
    widget = Retina.Widget.extend({
        about: {
                title: "AWE Workflow Editor",
                name: "awe_workflow",
                author: "Tobias Paczian",
                requires: [ ]
        }
    });
    
    widget.setup = function () {
	    return [ ];
    };

    widget.tables = [];
    widget.api_workflows = {};
        
    widget.display = function (wparams) {
	// initialize
        widget = this;
	var index = widget.index;

	widget.playground = wparams.playground;
	var content = wparams.target;

	// get workflows from the workflow api
	jQuery.getJSON(RetinaConfig.workflow_ip, function(data) {
	    var option_string = "";
	    for (i=0;i<data.data.length;i++) {
		widget.api_workflows[data.data[i].workflow_info.name] = data.data[i];
		option_string += '<option title="'+data.data[i].workflow_info.description+'">'+data.data[i].workflow_info.name+'</option>';
	    }
	    document.getElementById('api_workflows').innerHTML = option_string;
	});

	// workflow info
	content.innerHTML = '\
<form class="form-horizontal" style="float: left; margin-right: 50px;" action="#">\
\
    <legend>General Information</legend>\
\
    <h3>Workflow Info</h3>\
    <div class="control-group">\
      <label class="control-label" for="wf_name">name</label>\
      <div class="controls">\
        <input type="text" id="wf_name" placeholder="unique workflow name">\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="wf_author">author</label>\
      <div class="controls">\
        <input type="text" id="wf_author" placeholder="author name">\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="wf_contact">contact</label>\
      <div class="controls">\
        <input type="text" id="wf_contact" placeholder="user@host.com">\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="wf_description">description</label>\
      <div class="controls">\
        <input type="text" id="wf_description" placeholder="a short workflow summary">\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="wf_splits">splits</label>\
      <div class="controls">\
        <input type="text" id="wf_splits" placeholder="number of splits">\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="wf_infiles">#-input files</label>\
      <div class="controls">\
        <input type="text" id="wf_infiles" placeholder="number of input files">\
      </div>\
    </div>\
</form>';

	// variables
	content.innerHTML += '\
<form class="form-horizontal" style="float: left; position: relative; top: 65px;" action="#">\
\
    <h3>Variables</h3>\
    <div class="control-group">\
      <label class="control-label" for="v_name">name</label>\
      <div class="controls">\
        <input type="text" id="v_name" placeholder="variable name">\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="v_default">default</label>\
      <div class="controls">\
        <input type="text" id="v_default" placeholder="default value">\
      </div>\
    </div>\
\
    <button class="btn" style="float: right; top: -4px;" id="variable_add">add</button>\
    <button class="btn" style="left: 160px; position: relative;" id="variable_remove">remove</button>\
    <select id="variable_list" size=5 style="float: right; margin-top: 15px; clear: both;"></select>\
</form>';

	// task builder
	content.innerHTML += '\
<form class="form-horizontal" style="float: left; clear: left; margin-right: 50px;" action="#">\
\
    <legend id="task_legend">New Task</legend>\
\
    <div class="control-group">\
      <label class="control-label" for="task_command">command</label>\
      <div class="controls">\
        <input type="text" id="task_command" placeholder="script name">\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="task_args">arguments</label>\
      <div class="controls">\
        <input type="text" id="task_args" placeholder="-arg1=value1 -arg2=value2">\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="task_depends">depends on</label>\
      <div class="controls">\
        <select id="task_depends" size=5 multiple></select>\
      </div>\
    </div>\
</form>';

	// i/o
	content.innerHTML += '\
<form class="form-horizontal" style="float: left; position: relative; top: 65px;" action="#">\
\
    <div class="control-group">\
      <label class="control-label" for="task_input">input</label>\
      <div class="controls">\
        <select size=5 multiple id="task_input"></select>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="task_output">output</label>\
      <div class="controls">\
        <input type="text" id="new_output" placeholder="output.file"><br>\
        <button class="btn" style="float: right; margin-bottom: 15px; margin-top: 15px;" id="output_add">add</button>\
        <button class="btn" style="float: left; margin-bottom: 15px; margin-top: 15px;" id="output_remove">remove</button><br>\
        <select size=5 multiple id="task_output"></select>\
      </div>\
    </div>\
    <button class="btn" id="add_task_button">add task</button><button class="btn" id="edit_cancel_button" style="display: none;">cancel</button>\
</form>\
</div>';

	// visual / textual toggle
	content.innerHTML += '<div style="position: absolute; top: -15px; right: -90px;"><button type="button" class="btn" data-toggle="button" id="edit_active" onclick="document.getElementById(\'visual_button\').click();">edit task</button></div><div style="position: absolute; top: -15px; right: -720px;" class="btn-group" data-toggle="buttons-radio">\
    <button type="button" class="btn active" onclick="document.getElementById(\'playground\').style.display=\'none\';document.getElementById(\'json_holder\').style.display=\'\';">JSON</button>\
    <button type="button" class="btn" onclick="document.getElementById(\'json_holder\').style.display=\'none\';document.getElementById(\'playground\').style.display=\'\';" id="visual_button">visual</button>\
    </div>';

	// output JSON
	content.innerHTML += '<div style="width: 700px; height: 750px; position: absolute; right: -730px;padding: 10px;top: 13px;" id="json_holder"><textarea style="width: 690px; height: 690px;" id="json_out">{ }</textarea><button class="btn" style="float: left;" id="result_validate">validate</button><input type="file" style="float: left;" id="workflow_load"><select id="api_workflows"></select><button class="btn" style="float: left;" id="load_workflow_api">load</button><button class="btn" style="float: right;" id="result_save" onclick="stm.saveAs(document.getElementById(\'json_out\').innerHTML, \'workflow.awf\');">save</button></div>';

	// event listeners
	document.getElementById('wf_name').addEventListener('change', function() {
	    if (widget.loading) { return }
	    widget.data.workflow_info.name = document.getElementById('wf_name').value;
	    widget.update();
	});

	document.getElementById('wf_author').addEventListener('change', function() {
	    if (widget.loading) { return }
	    widget.data.workflow_info.author = document.getElementById('wf_author').value;
	    widget.update();
	});

	document.getElementById('wf_contact').addEventListener('change', function() {
	    if (widget.loading) { return }
	    widget.data.workflow_info.contact = document.getElementById('wf_contact').value;
	    widget.update();
	});

	document.getElementById('wf_description').addEventListener('change', function() {
	    if (widget.loading) { return }
	    widget.data.workflow_info.description = document.getElementById('wf_description').value;
	    widget.update();
	});

	document.getElementById('wf_splits').addEventListener('change', function() {
	    if (widget.loading) { return }
	    var splits = parseInt(document.getElementById('wf_splits').value);
	    if (! splits) {
		alert("splits must be an integer value");
		document.getElementById('wf_splits').value = "";
	    } else {
		widget.data.job_info.splits = parseInt(document.getElementById('wf_splits').value);
		widget.update();
	    }
	});

	document.getElementById('wf_infiles').addEventListener('change', function() {
	    if (widget.loading) { return }
	    var infiles = parseInt(document.getElementById('wf_infiles').value);
	    if (! infiles) {
		alert("number of input files must be an integer value");
		document.getElementById('wf_infiles').value = "";
	    } else {
		widget.data.raw_inputs = {};
		var newinputs = [];
		for (i=0;i<widget.inputs.length;i++) {
		    if (! widget.inputs[i].match(/^#i_\d+ \[0\]$/)) {
			newinputs.push(widget.inputs[i]);
		    }
		}
		widget.inputs = newinputs;
		for (i=0;i<infiles;i++) {
		    widget.data.raw_inputs["#i_"+(i+1)] = "#data_url";
		    widget.inputs.push("#i_"+(i+1)+" [0]");
		}
		widget.update();
		widget.new_task();
	    }
	});

	document.getElementById('variable_add').addEventListener('click', function() {
	    if ((document.getElementById('v_name').value.length > 0) && (document.getElementById('v_default').value.length > 0)) {
		widget.data.variables[document.getElementById('v_name').value] = document.getElementById('v_default').value;
		var vl = document.getElementById('variable_list');
		vl.options.length = 0;
		for (i in widget.data.variables) {
		    if (widget.data.variables.hasOwnProperty(i)) {
			vl.options[vl.options.length] = new Option(i,i);
		    }
		}
		document.getElementById('v_name').value = "";
		document.getElementById('v_default').value = "";
		widget.update();
	    }
	});

	document.getElementById('variable_remove').addEventListener('click', function() {
	    var vl = document.getElementById('variable_list');
	    if (vl.selectedIndex > -1) {
		delete widget.data.variables[vl.options[vl.selectedIndex].value];
		vl.options.length = 0;
		for (i in widget.data.variables) {
		    if (widget.data.variables.hasOwnProperty(i)) {
			vl.options[vl.options.length] = new Option(i,i);
		    }
		}
		widget.update();
	    }
	});

	document.getElementById('output_add').addEventListener('click', function() {
	    if (document.getElementById('new_output').value.length > 0) {
		var vl = document.getElementById('task_output');
		vl.options[vl.options.length] = new Option(document.getElementById('new_output').value, document.getElementById('new_output').value);
		document.getElementById('new_output').value = "";
	    }
	});

	document.getElementById('output_remove').addEventListener('click', function() {
	    var vl = document.getElementById('task_output');
	    if (vl.selectedIndex > -1) {
		vl.options[vl.selectedIndex] = null;
	    }
	});

	document.getElementById('add_task_button').addEventListener('click', function() {
	    if (document.getElementById('task_legend').innerHTML == "New Task") {
		widget.new_task();
	    } else {
		widget.edit_task();
	    }
	});

	document.getElementById('result_save').addEventListener('click', function() {
	    widget.save();
	});

	document.getElementById('result_validate').addEventListener('click', function() {
	    widget.validate();
	});

	document.getElementById('workflow_load').addEventListener('change', function() {
	    widget.load_workflow();
	});

	document.getElementById('json_out').addEventListener('click', function(ev) {
	    ev = ev || window.event;
	    console.log(ev);
	});

	document.getElementById('load_workflow_api').addEventListener('click', function() {
	    if (document.getElementById('api_workflows').options.length > 0) {
		var selected = document.getElementById('api_workflows').options[document.getElementById('api_workflows').selectedIndex].value;
		if (widget.api_workflows.hasOwnProperty(selected)) {
		    widget.data = widget.api_workflows[selected];
		    widget.parse_workflow();
		}
	    }
	});

	widget.update();
	widget.new_task();
    };

    widget.validate = function () {

    };

    widget.edit_task = function () {
	alert('editing');
    }

    widget.parse_workflow = function () {
	widget.loading = 1;
	widget.data.workflow_info.splits = 8;
	document.getElementById('wf_name').value = widget.data.workflow_info.name;
	document.getElementById('wf_author').value = widget.data.workflow_info.author;
	document.getElementById('wf_contact').value = widget.data.workflow_info.contact;
	document.getElementById('wf_description').value = widget.data.workflow_info.description;
	document.getElementById('wf_splits').value = widget.data.workflow_info.splits;
	var vl = document.getElementById('variable_list');
	vl.options.length = 0;
	for (i in widget.data.variables) {
	    if (widget.data.variables.hasOwnProperty(i)) {
		vl.options[vl.options.length] = new Option(i,i);
	    }
	}
	var numfiles = 0;
	widget.inputs = [];
	for (i in widget.data.raw_inputs) {
	    if (widget.data.raw_inputs.hasOwnProperty(i)) {
		numfiles++;
		widget.inputs.push("#i_"+numfiles);
	    }
	}
	document.getElementById('wf_infiles').value = numfiles;
	widget.tasks = { 0: 1 };
	for (x=0;x<widget.data.tasks.length;x++) {
	    widget.tasks[widget.data.tasks[x].taskid] = 1;
	    for (h=0;h<widget.data.tasks[x].outputs.length;h++) {
		widget.inputs.push(widget.data.tasks[x].outputs[h] +" ["+widget.data.tasks[x].taskid+"]");
	    }
	    widget.add_box(widget.data.tasks[x]);
	}
	widget.loading = 0;
	
	var dep = document.getElementById('task_depends');
	var tin = document.getElementById('task_input');
	tin.options.length = 0;
	for (i=0; i<widget.inputs.length; i++) {
	    tin.options[tin.options.length] = new Option(widget.inputs[i],widget.inputs[i]);
	}
	
	dep.options.length = 0;
	for (i in widget.tasks) {
	    if (widget.tasks.hasOwnProperty(i)) {
		dep.options[dep.options.length] = new Option(i,i);
	    }
	}
	widget.update();
    }
    
    widget.load_workflow = function () {
	var file = document.getElementById('workflow_load').files[0];
	var fileReader = new FileReader();
	fileReader.onload = (function(ev){
	    widget.data = JSON.parse(ev.target.result);
	    widget.parse_workflow();
	});
	fileReader.readAsText(file);
    };

    widget.update = function () {
	var now = new Date();
	var datum = now.getFullYear() + "-" + (now.getMonth() < 10 ? "0" : "") + now.getMonth() + "-" + (now.getDate() < 10 ? "0" : "") + now.getDate();
	widget.data.workflow_info.update_date = datum;
	document.getElementById('json_out').innerHTML = JSON.stringify(widget.data, null, 5);
    }

    widget.new_task = function () {
	var dep = document.getElementById('task_depends');
	var tin = document.getElementById('task_input');
	var out = document.getElementById('task_output');

	if (document.getElementById('task_command').value.length) {
	    var new_task = {};
	    new_task.taskid = widget.data.tasks.length + 1;
	    new_task.cmd = { "name": document.getElementById('task_command').value,
			     "args": document.getElementById('task_args').value || "" };
	    var deps = [];
	    for (i=0;i<dep.options.length;i++) {
		if (dep.options[i].selected) {
		    deps.push(dep.options[i].value);
		}
	    }
	    new_task.dependsOn = deps;

	    var inps = {};
	    for (i=0;i<tin.options.length;i++) {
		if (tin.options[i].selected) {
		    var val = tin.options[i].value;
		    var num = val.substring(val.lastIndexOf('[') + 1, val.lastIndexOf(']'));
		    var fn = val.substring(0, val.lastIndexOf('[') - 1);
		    inps[fn] = num;
		}
	    }
	    new_task.inputs = inps;

	    var outs = [];
	    for (i=0;i<out.options.length;i++) {
		outs.push(out.options[i].value);
		widget.inputs.push(out.options[i].value +" ["+new_task.taskid+"]");
	    }
	    new_task.outputs = outs;
	    
	    document.getElementById('task_args').value = "";
	    document.getElementById('task_command').value = "";
	    out.options.length = 0;
	    widget.tasks[new_task.taskid] = 1;

	    widget.data.tasks.push(new_task);

	    // playground
	    widget.add_box(new_task);

	    widget.update();

	}
	
	tin.options.length = 0;
	for (i=0; i<widget.inputs.length; i++) {
	    tin.options[tin.options.length] = new Option(widget.inputs[i],widget.inputs[i]);
	}

	dep.options.length = 0;
	for (i in widget.tasks) {
	    if (widget.tasks.hasOwnProperty(i)) {
		dep.options[dep.options.length] = new Option(i,i);
	    }
	}
    };

    widget.inputs = [ "#i_1 [0]" ];

    widget.tasks = { 0: 1 };

    widget.data = {
        "workflow_info":{
            "name":"",
            "author":"",
            "contact":"",
            "update_date":"",
            "description":""
        },
        "job_info":{
            "jobname": "#default_jobname",
            "project": "#default_project",
            "user": "#default_user",
            "queue": "#default_queue",
            "splits":8
        },
        "raw_inputs":{
            "#i_1":"#data_url",
        },
        "data_server":"#shock_host",
        "variables":{
        },
        "tasks": [
        ]
    };
    
    // visual editor
    widget.curr_box_pos = { x: 10, y: 10 };
    widget.box_size = 100;
    widget.playground_width = 699;
    widget.playground_height = 749;
    widget.box_padding = 20;

    widget.add_box = function (task) {
	var box = document.createElement('div');
	box.setAttribute('class', 'taskbox');
	box.setAttribute('id', 'taskbox'+task.taskid);
	box.boxid = task.taskid;
	box.widget = this.index;
	box.addEventListener('click', function () {
	    if (document.getElementById('edit_active').className == "btn active") {
		var tasks = Retina.WidgetInstances.awe_workflow[this.widget].data.tasks;
		var thistask;
		for (i=0;i<tasks.length;i++) {
		    if (tasks[i].taskid == this.boxid) {
			thistask = tasks[i];
			break;
		    }
		}
		var tdeps = {};
		for (i=0;i<thistask.dependsOn.length;i++) {
		    tdeps[thistask.dependsOn[i]] = true;
		}
		var touts = {};
		for (i=0;i<thistask.outputs.length;i++) {
		    touts[thistask.outputs[i]] = true;
		}
		document.getElementById('task_legend').innerHTML = "Edit Task "+this.boxid;
		document.getElementById('add_task_button').innerHTML = "save changes";
		document.getElementById('edit_cancel_button').style.display = "";
		document.getElementById('task_command').value = thistask.cmd.name;
		document.getElementById('task_args').value = thistask.cmd.args;
		var tdep = document.getElementById('task_depends');
		for (i=0;i<tdep.options.length;i++) {
		    if (tdeps[tdep.options[i].value]) {
			tdep.options[i].selected = true;
		    } else {
			tdep.options[i].selected = false;
		    }
		}
		var tinp = document.getElementById('task_input');
		for (i=0;i<tinp.options.length;i++) {
		    if (thistask.inputs.hasOwnProperty(tdep.options[i].value)) {
			tinp.options[i].selected = true;
		    } else {
			tinp.options[i].selected = false;
		    }
		}
		var tout = document.getElementById('task_output');
		for (i=0;i<tout.options.length;i++) {
		    if (touts[tout.options[i].value]) {
			tout.options[i].selected = true;
		    } else {
			tout.options[i].selected = false;
		    }
		}
	    }
	});
	box.addEventListener('mouseover', function () {
	    var tasks = Retina.WidgetInstances.awe_workflow[this.widget].data.tasks;
	    var thistask;
	    for (i=0;i<tasks.length;i++) {
		if (tasks[i].taskid == this.boxid) {
		    thistask = i;
		}
		for (h=0;h<tasks[i].dependsOn.length;h++) {
		    if (tasks[i].dependsOn[h] == this.boxid) {
			document.getElementById('taskbox'+(i+1)).className = "taskbox dependor";
		    }
		}
	    }
	    for (i=0;i<tasks[thistask].dependsOn.length;i++) {
		if (tasks[thistask].dependsOn[i] == 0) {
		    continue;
		}
		document.getElementById('taskbox'+tasks[thistask].dependsOn[i]).className = "taskbox dependant";
	    }
	});
	box.addEventListener('mouseout', function () {
	    var tasks = Retina.WidgetInstances.awe_workflow[this.widget].data.tasks;
	    for (i=0;i<tasks.length;i++) {
		document.getElementById('taskbox'+(i+1)).className = "taskbox";
	    }
	});
	box.setAttribute('style', 'top: '+widget.curr_box_pos.y+'px; left: '+widget.curr_box_pos.x+'px;');
	if ((widget.curr_box_pos.x + (widget.box_size * 2) + (widget.box_padding * 2)) > widget.playground_width) {
	    widget.curr_box_pos.x = (widget.box_padding / 2);
	    widget.curr_box_pos.y += (widget.box_padding * 2) + widget.box_size;
	} else {
	    widget.curr_box_pos.x += widget.box_size + (widget.box_padding * 2);
	}
	box.innerHTML = "<h3 style='width: 100%; text-align: center;'>"+task.taskid+"</h3><p style='width: 100%; text-align: center; word-wrap: break-word;'>"+task.cmd.name+"</p>";

	// inputs
	var inp_top = 20;
	for (i in task.inputs) {
	    if (task.inputs.hasOwnProperty(i)) {
		var inp = document.createElement('div');
		inp.setAttribute('style', "width: 10px; height: 10px; border: 1px solid black; background-color: yellow; position: absolute; bottom: "+inp_top+"px; left: 0px;");
		inp.setAttribute('title', i+" ["+task.inputs[i]+"]");
		box.appendChild(inp);
		inp_top += 15;
	    }
	}

	// outputs
	var out_top = 20;
	for (i=0;i<task.outputs.length;i++) {
	    var out = document.createElement('div');
	    out.setAttribute('style', "width: 10px; height: 10px; border: 1px solid black; background-color: blue; position: absolute; bottom: "+out_top+"px; right: 0px;");
	    out.setAttribute('title', task.outputs[i]);
	    box.appendChild(out);
	    out_top += 15;
	}

	// dependencies
	var dep_left = 20;
	for (i=0;i<task.dependsOn.length;i++) {
	    var dep = document.createElement('div');
	    dep.setAttribute('style', "width: 10px; height: 10px; border: 1px solid black; background-color: red; position: absolute; left: "+dep_left+"px; bottom: 5px;");
	    dep.setAttribute('title', task.dependsOn[i]);
	    box.appendChild(dep);
	    dep_left += 15;
	}

	box.addEventListener('mousedown', function (ev) {
	    ev = ev || window.event;
	    widget.box = { x: ev.clientX,
			   y: ev.clientY,
			   xorig: parseInt(this.style.left) || 0,
			   yorig: parseInt(this.style.top) || 0,
			   elem: this };
	    widget.moving = true;
	});
	widget.playground.appendChild(box);
    };

    widget.end_move = function () {
	if (widget.box) {
	    if (parseInt(widget.box.elem.style.top)<0) {
		widget.box.elem.style.top = "0px";
	    }
	    if (parseInt(widget.box.elem.style.top)>649) {
		widget.box.elem.style.top = "649px";
	    }
	    if (parseInt(widget.box.elem.style.left)<0) {
		widget.box.elem.style.left = "0px";
	    }
	    if (parseInt(widget.box.elem.style.left)>599) {
		widget.box.elem.style.left = "599px";
	    }
	}
    };

    window.addEventListener('mouseup', function (){
	widget.moving = false;
	widget.end_move();
    });

    window.addEventListener('mousemove', function (ev) {
	if (widget.moving) {
	    ev = ev || window.event;
	    var y = ev.clientY - widget.box.y;
	    var x = ev.clientX - widget.box.x;
	    widget.box.elem.style.top = (widget.box.yorig + y) + "px";
	    widget.box.elem.style.left = (widget.box.xorig + x) + "px";
	}
    });

})();
