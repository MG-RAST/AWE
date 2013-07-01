(function () {
    widget = Retina.Widget.extend({
        about: {
                title: "AWE Job Submission",
                name: "awe_submission",
                author: "Tobias Paczian",
                requires: [ ]
        }
    });
    
    widget.setup = function () {
	    return [ Retina.add_renderer({"name": "table", "resource": "./renderers/",  "filename": "renderer.table.js" }),
  		     Retina.load_renderer("table") ];
    };

    widget.tables = [];
        
    widget.display = function (wparams) {
	// initialize
        widget = this;
	var index = widget.index;

	var html = '\
<form class="form-horizontal">\
\
    <legend>Submit Job</legend>\
\
    <div class="control-group">\
      <label class="control-label" for="shock_url">SHOCK URL</label>\
      <div class="controls">\
        <input type="text" id="shock_url" placeholder="SHOCK IP:port">\
        <span class="help-block">Enter the SHOCK IP and port, e.g. localhost:1234</span>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="input_file">input file</label>\
      <div class="controls">\
        <input id="input_file" type="file">\
        <span class="help-block">Select the input file for your job</span>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="pipeline">pipeline</label>\
      <div class="controls">\
        <select id="pipeline">\
          <option>Pipeline A</option>\
          <option>Pipeline B</option>\
          <option>Pipeline C</option>\
          <option>Pipeline D</option>\
        </select>\
        <span class="help-block">Select which pipeline to run</span>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="job_name">job name</label>\
      <div class="controls">\
        <input type="text" placeholder="job name" id="job_name">\
        <span class="help-block">Enter a name to identify your submitted job.</span>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="user_name">user name</label>\
      <div class="controls">\
        <input type="text" placeholder="user name" id="user_name">\
        <span class="help-block">Enter the username running the job.</span>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="group_name">group name</label>\
      <div class="controls">\
        <input type="text" placeholder="group name" id="group_name">\
        <span class="help-block">Enter the name of the group to which the job belongs.</span>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="project_name">project name</label>\
      <div class="controls">\
        <input type="text" placeholder="project name" id="project_name">\
        <span class="help-block">Enter the name of the project of this job.</span>\
      </div>\
    </div>\
\
    <button class="btn btn-success">Submit Job</button>\
</form>';

	// awe_submit.pl -awe=localhost:8000 -shock=<shock_url> -upload=<file_path> -pipeline=<path of seleted pipeline template> -name=<string> -project=<string> -user=<string> -cgroup=<string> -totalwork=<a configurable int>

	document.getElementById('content').innerHTML = html;
    };    
})();