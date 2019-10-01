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
    widget.shock_checked = false;
        
    widget.display = function (wparams) {
	// initialize
        widget = this;
	var index = widget.index;

	var html = '\
<form class="form-horizontal" action="#">\
\
    <legend>Submit Job</legend>\
\
    <div class="control-group">\
      <label class="control-label" for="input_file" style="position: relative; bottom: -64px;">input file</label>\
      <div class="controls">\
        <ul class="nav nav-tabs">\
          <li class="active"><a href="#newfile" data-toggle="tab">upload new</a></li>\
          <li><a href="#existingfile" data-toggle="tab">browse SHOCK</a></li>\
        </ul>\
        <div class="tab-content">\
          <div class="tab-pane active" id="newfile">\
            <input id="input_file" type="file" style="margin-left: -10px;">\
          </div>\
          <div class="tab-pane" id="existingfile">\
            <select style="margin-left: -10px;">\
              <option>file a</option>\
              <option>file b</option>\
              <option>file c</option>\
              <option>file d</option>\
              <option>file e</option>\
              <option>file f</option>\
              <option>file g</option>\
              <option>file h</option>\
              <option>file i</option>\
              <option>file j</option>\
            </select>\
          </div>\
        </div>\
        <span class="help-block">Select the input file for your job</span>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="shock_url">SHOCK URL</label>\
      <div class="controls">\
        <input type="text" id="shock_url" placeholder="SHOCK IP:port" value="'+(RetinaConfig.shock_ip || "")+'" style="border-radius: 3px 0 0 3px;">\
        <button class="btn" onclick="Retina.WidgetInstances.awe_submission[1].check_shock();" style="border-radius: 0 3px 3px 0; margin-left: -5px;">verify</button>\
        <span class="help-block">Enter the SHOCK IP and port, e.g. localhost:1234</span>\
      </div>\
    </div>\
\
    <div class="control-group">\
      <label class="control-label" for="pipeline">pipeline</label>\
      <div class="controls">\
        <select id="pipeline">\
          <option>mgrast-prod</option>\
          <option>mgrast-v3</option>\
          <option>mgrast.json</option>\
          <option>simple_example</option>\
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

	document.getElementById('content').innerHTML = html;
    };

    widget.submit_jobscript = function (jobscript) {
	var fd = new FormData();
	var oMyBlob = new Blob([jobscript], { "type" : "text/json" });
	fd.append("upload", oMyBlob);
	jQuery.ajax(RetinaConfig.awe_url+"/job", {
	    contentType: false,
	    processData: false,
	    data: fd,
	    success: function(data) {
		console.log(data);
	    },
	    error: function(jqXHR, error){
		console.log(error);
	    },
	    type: "POST"
	});
    };

    widget.check_shock = function (ev) {
	var url = "http://"+document.getElementById('shock_url').value;
	jQuery.ajax(url, {
	    success: function(data) {
		if (data && data.type && data.type == "Shock") {
		    Retina.WidgetInstances.awe_submission[1].shock_checked = true;
		    SHOCK.init({ "url": url });
		    alert('successfully connected to SHOCK server');
		} else {
		    alert('could not identify target as SHOCK server');
		}
	    },
	    error: function(jqXHR, error) {
		alert('could not reach server: '+error);
	    },
	    type: "GET"
	});
    };

    widget.get_template = function (template_name ) {
	
    };

})();
