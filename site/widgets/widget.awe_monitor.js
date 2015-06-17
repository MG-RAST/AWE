(function () {
    widget = Retina.Widget.extend({
        about: {
                title: "AWE Task Status Monitor",
                name: "awe_monitor",
                author: "Tobias Paczian",
                requires: [ 'xlsx.js', 'jszip.min.js' ]
        }
    });
    
    widget.setup = function () {
	stm.DataStore.job = {};
	return [ Retina.load_renderer("table") ];
    };

    widget.tables = [];
        
    widget.display = function (wparams) {
        widget = this;
	var index = widget.index;

	jQuery.ajax({ url: RetinaConfig["awe_ip"],
		      dataType: "json",
		      success: function(data) {
			  if (data.hasOwnProperty('title')) {
			      document.getElementById('serverTitle').innerHTML = data.title;
			  }
		      }
		    });

	widget.target = widget.target || wparams.target;
	widget.target.innerHTML = '\
<div id="refresh" style="position: absolute; top: 64px; left: 17px; z-index: 100;">\
    </div>\
    <div id="overview" class="sidebar"></div>\
    <div class="mainview">\
      <ul class="nav nav-tabs">\
        <li class="active">\
	  <a data-toggle="tab" href="#jobs">Jobs</a>\
	</li>\
	<li>\
	  <a data-toggle="tab" href="#workunits">Workunits</a>\
	</li>\
	<li>\
	  <a data-toggle="tab" href="#clients">Clients</a>\
	</li>\
	<li>\
	  <a data-toggle="tab" href="#debug" id="debugRef">Debug</a>\
	</li>\
      </ul>\
      <div class="tab-content">\
	<div id="jobs" class="tab-pane active">\
	</div>\
	<div id="workunits" class="tab-pane">\
	</div>\
	<div id="clients" class="tab-pane">\
	</div>\
	<div id="debug" class="tab-pane" style="width: 800px;">\
	</div>\
      </div>\
    </div>';

	var update = document.getElementById('refresh');
	update.innerHTML = '\
<div class="alert alert-block alert-info" style="width: 235px;">\
  <button type="button" class="close" data-dismiss="alert">×</button>\
  <h4>updating...</h4>\
  <div class="progress progress-striped active" style="margin-bottom: 0px; margin-top: 10px;">\
    <div class="bar" style="width: 0%;" id="pbar"></div>\
  </div>\
</div>';

	widget.updated = 0;
	Retina.RendererInstances.table = [ Retina.RendererInstances.table[0] ];

	var views = [ 
	    "overview",
	    "jobs",
	    "workunits",
	    "clients" ];

	for (i=0;i<views.length;i++) {
	    var view = document.getElementById(views[i]);
	    view.innerHTML = "";
	   
	    if (views[i] != "overview") {
		var options = document.createElement('div');
		options.setAttribute('id', 'optionsDiv'+views[i]);
		view.appendChild(options);
		var target_space = document.createElement('div');
		view.appendChild(target_space);
		Retina.WidgetInstances.awe_monitor[1].tables[views[i]] = Retina.Renderer.create("table", { target: target_space, data: {}, filter_autodetect: true, sort_autodetect: true });
	    }
	    
	    widget.update_data(views[i]);
	}
    };

    widget.update_data = function (which) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var return_data = {};

	switch (which) {
	case "overview":
	    jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/queue",
			   headers: widget.authHeader,
			   error: function () {
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   },
			   success: function(data) {
			       var widget = Retina.WidgetInstances.awe_monitor[1];
			       var result = data.data;
			       var rows = result.split("\n");
			       
			       return_data = { "total jobs": { "all": rows[1].match(/\d+/)[0],
							       "in-progress": rows[2].match(/\d+/)[0],
							       "suspended": rows[3].match(/\d+/)[0] },
					       "total tasks": { "all": rows[4].match(/\d+/)[0],
								"queuing": rows[5].match(/\d+/)[0],
								"in-progress": rows[6].match(/\d+/)[0],
								"pending": rows[7].match(/\d+/)[0],
								"completed": rows[8].match(/\d+/)[0],
								"suspended": rows[9].match(/\d+/)[0] },
					       "total workunits": { "all": rows[11].match(/\d+/)[0],
								    "queueing": rows[12].match(/\d+/)[0],
								    "checkout": rows[13].match(/\d+/)[0],
								    "suspended": rows[14].match(/\d+/)[0] },
					       "total clients": { "all": rows[15].match(/\d+/)[0],
								  "busy": rows[16].match(/\d+/)[0],
								  "idle": rows[17].match(/\d+/)[0],
								  "suspended": rows[18].match(/\d+/)[0] } };
			       
			       var html = '<h4>Overview</h4><table class="table">';
			       for (h in return_data) {
				   if (return_data.hasOwnProperty(h)) {
				       html += '<tr><th colspan=2>'+h+'</th><th>'+return_data[h]['all']+'</th></tr>';
				       for (j in return_data[h]) {
					   if (return_data[h].hasOwnProperty(j)) {
					       if (j == 'all') {
						   continue;
					       }
					       html += '<tr><td></td><td>'+j+'</td><td>'+return_data[h][j]+'</td></tr>';
					   }
				       }
				   }
			       }
			       html += '</table>';
			       
			       Retina.WidgetInstances.awe_monitor[1].check_update();

			       document.getElementById('overview').innerHTML = html;
			   }
			 });
	    return;

	    break;
	case "jobs":
	    var gt = Retina.WidgetInstances.awe_monitor[1].tables["jobs"];
	    gt.settings.headers = widget.authHeader;
	    gt.settings.synchronous = false;
	    gt.settings.query_type = 'prefix';
	    gt.settings.data_manipulation = Retina.WidgetInstances.awe_monitor[1].dataManipulationJobs,
	    gt.settings.navigation_url = RetinaConfig["awe_ip"]+"/job?query";
	    gt.settings.rows_per_page = 20;
	    gt.settings.minwidths = [1,150,150,1, 95, 125];
	    gt.settings.disable_sort = { 3: 1 };
	    gt.settings.filter = { 1: { type: "text" },
				   2: { type: "text" },
				   4: { type: "text" },
				   5: { type: "premade-select", options: [
				       { "text": "show all", "value": "" },
				       { "text": "completed", "value": "completed" },
				       { "text": "suspend", "value": "suspend" },
				       { "text": "in-progress", "value": "in-progress" },
				       { "text": "checkout", "value": "checkout" },
				       { "text": "queued", "value": "queued" }
				   ] } };
	    gt.settings.asynch_column_mapping = { "submission": "info.submittime",
						  "job": "info.name",
						  "job id": "jid",
						  "pipeline": "info.pipeline",
						  "current state": "state" };
	    gt.settings.filter_autodetect = false;
	    gt.settings.sort_autodetect = false;
	    gt.settings.data = { data: [], header: [ "submission", "job name", "job id", "status", "pipeline", "current state" ] };
	    gt.render();
	    gt.update({}, gt.index);

	    break;
	case "workunits":
	    var qwt = Retina.WidgetInstances.awe_monitor[1].tables["workunits"];
	    qwt.settings.headers = widget.authHeader;
	    qwt.settings.synchronous = false;
	    qwt.settings.query_type = 'prefix';
	    qwt.settings.data_manipulation = Retina.WidgetInstances.awe_monitor[1].dataManipulationWorkunits,
	    qwt.settings.navigation_url = RetinaConfig["awe_ip"]+"/work?query";
	    qwt.settings.rows_per_page = 10;
	    qwt.settings.minwidths = [1,1,1,1,65,78,75,75,83];
	    qwt.settings.asynch_column_mapping = { "wuid": "wuid",
						   "submission time": "info.submittime",
						   "cmd name": "cmd.name",
						   "cmd args": "cmd.args",
						   "rank": "rank",
						   "tot": "totalwork",
						   "state": "state",
						   "failed": "failed",
						   "priority": "info.priority" };
	    qwt.settings.filter = { 0: { type: "text" },
				    1: { type: "text" },
				    2: { type: "text" },
				    3: { type: "text" },
				    4: { type: "text" },
				    5: { type: "text" },
				    6: { type: "text" },
				    7: { type: "text" },
				    8: { type: "text" } };
	    qwt.settings.disable_sort = {};
	    qwt.settings.filter_autodetect = false;
	    qwt.settings.sort_autodetect = false;
	    qwt.settings.data = { data: [], header: [ "wuid",
						      "submission time",
						      "cmd name",
						      "cmd args",
						      "rank",
						      "tot",
						      "state",
						      "failed",
						      "priority" ] };
	    qwt.render();
	    qwt.update({}, qwt.index);
	    break;
	case "clients":
	    var options = document.getElementById('optionsDivclients');
	    options.innerHTML = "<button class='btn btn-small btn-primary' onclick='Retina.WidgetInstances.awe_monitor[1].resumeAllClients();' style='margin-bottom: 10px;'>resume all clients</button>";
	    jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/client",
			   headers: widget.authHeader,
			   success: function (data) {
			       var widget = Retina.WidgetInstances.awe_monitor[1];
			       var result_data = [];
			       if (data.data == null) {
				   result_data = [ ['-','-','-','-','-','-','-','-','-','-','-','-','-','-','-'] ];
			       } else {
				   for (var h=0;h<data.data.length;h++) {
				       var obj = data.data[h];
				       var skipwork = [];
				       for (var j=0;j<obj.skip_work.length;j++) {
					   skipwork.push("<a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].workunitTooltip(jQuery(this), \""+obj.skip_work[j]+"\")'>"+(j+1)+"</a>");
				       }
				       result_data.push( [ "<a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].clientTooltip(jQuery(this), \""+obj.id+"\")'>"+obj.name+"</a>",
							  obj.group,
							  obj.host,
							  obj.cores || "0",
							  obj.apps.join(", "),
							  obj.regtime,
							  obj.serve_time,
							  obj.subclients || "0",
							  obj.Status,
							  obj.total_checkout || "0",
							  obj.total_completed || "0",
							  obj.total_failed || "0",
							  skipwork.join(", ") ]);
				   }
			       }
			       return_data = { header: [ "name",
							 "group",
							 "host",
							 "cores",
							 "apps",
							 "register time",
							 "up-time",
							 "subclients",
							 "status",
							 "c/o",
							 "done",
							 "failed",
							 "errors"],
					       data: result_data };

			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.rows_per_page = 15;
			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.invisible_columns = { 4: true };
			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.minwidths = [110,73,70,73,75,115,90,105,75,60,70,75,90];
			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.data = return_data;
			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].render();
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   }
	    });

	    break;
	default:
	    return null;
	}
    };
    
    widget.check_update = function () {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	Retina.WidgetInstances.awe_monitor[1].updated += 100 / 2;
	if (parseInt(Retina.WidgetInstances.awe_monitor[1].updated) == 100) {
	    document.getElementById('refresh').innerHTML = '<button class="btn" onclick="Retina.WidgetInstances.awe_monitor[1].display();">refresh</button>';
	} else {
	    document.getElementById('pbar') ? document.getElementById('pbar').setAttribute('style', "width: "+Retina.WidgetInstances.awe_monitor[1].updated+"%;") : "";
	}
    };

    widget.dots = function (stages) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var dots = '<span>';
	if (stages.length > 0) {
	    for (var i=0;i<stages.length;i++) {
		if (stages[i].state == 'completed') {
		    dots += '<span style="color: green;font-size: 19px; cursor: default;" title="completed: '+stages[i].cmd.description+'">&#9679;</span>';
		} else if (stages[i].state == 'in-progress') {
		    dots += '<span style="color: blue;font-size: 19px; cursor: default;" title="in-progress: '+stages[i].cmd.description+'">&#9679;</span>';
		} else if (stages[i].state == 'queued') {
		    dots += '<span style="color: orange;font-size: 19px; cursor: default;" title="queued: '+stages[i].cmd.description+'">&#9679;</span>';
		} else if (stages[i].state == 'error' || stages[i].state == 'suspend') {
		    dots += '<span style="color: red;font-size: 19px; cursor: default;" title="error: '+stages[i].cmd.description+'">&#9679;</span>';
		} else if (stages[i].state == 'init' || stages[i].state == 'pending') {
		    dots += '<span style="color: gray;font-size: 19px; cursor: default;" title="init: '+stages[i].cmd.description+'">&#9679;</span>';
		} else {
		    console.log(stages[i].state);
		}
	    }
	}
			  
	dots += "</span>";

	return dots;
    };

    widget.tasksort = function (a, b) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var order = { "suspend": 0, "submitted": 1, "in-progress": 2 };
	if (order[a.state] > order[b.state]) {
	    return -1;
	} else if (order[a.state] < order[b.state]) {
	    return 1;
	} else {
	    return a.info.submittime.localeCompare(b.info.submittime);
	}
    };

    widget.jobTooltip = function (obj, id) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	obj.popover('destroy');
	obj.popover({content: "<button class='close' style='position: relative; bottom: 8px; left: 8px;' type='button' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>×</button><a style='cursor: pointer;' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);Retina.WidgetInstances.awe_monitor[1].jobDetails(\""+id+"\");'>job details</a><br><a style='cursor: pointer;' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);Retina.WidgetInstances.awe_monitor[1].jobDetails(\""+id+"\",true);'>job JSON</a><br><a style='cursor: pointer;' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/job/"+id+"?perf\");'>job stats</a>",html:true,placement:"top"});
	obj.popover('show');
    };

    widget.clientTooltip = function (obj, id) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	obj.popover('destroy');
	obj.popover({content: "<button class='close' style='position: relative; bottom: 8px; left: 8px;' type='button' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>×</button><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/client/"+id+"\");this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>details</a><br><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].resumeClient(&#39;"+id+"&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>resume</a>",html:true,placement:"top"});
	obj.popover('show');
    };


    widget.workunitTooltip = function (obj, wuid, jid) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	obj.popover('destroy');
	obj.popover({content: "<button class='close' style='position: relative; bottom: 8px; left: 8px;' type='button' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>×</button><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].workunitDetails(&#39;"+wuid+"&#39;,&#39;stderr&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>error</a><br><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].workunitDetails(&#39;"+wuid+"&#39;,&#39;stdout&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>output</a><br><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].workunitDetails(&#39;"+wuid+"&#39;,&#39;worknotes&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>worknotes</a><br><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].resumeJob(&#39;"+jid+"&#39;,&#39;worknotes&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>resume job</a>",html:true,placement:"top"});
	obj.popover('show');
    };

    widget.workunitDetails = function (id, which) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	if (which=='stdout') {
	    Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(RetinaConfig["awe_ip"]+"/work/"+id+"?report=stdout");
	} else if (which=='stderr') {
	    Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(RetinaConfig["awe_ip"]+"/work/"+id+"?report=stderr");
	} else if (which=='worknotes') {
	    Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(RetinaConfig["awe_ip"]+"/work/"+id+"?report=worknotes");
	} else {
	    console.log('call to workunitDetails for id "'+id+'" with unknown param: "'+which+'"');
	}
    };

    widget.jobDetails = function (jobid, json) {
	var widget = this;
	var job = stm.DataStore.job[jobid];
	if (json) {
	    stm.saveAs(JSON.stringify(job, null, 2), "job"+job.jid+".json");
	} else {
	    document.getElementById('debug').innerHTML = widget.stagePills(job);
	    document.getElementById('debugRef').click();
	}
    };

    /*
      Data Manipulation Functions (tables)
     */
    widget.dataManipulationWorkunits = function (data) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var result_data = [];
	for (var i=0;i<data.length;i++) {
	    var obj = data[i];
	    result_data.push( { "wuid": "<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/work/"+obj.wuid+"\");' style='cursor: pointer;'>"+obj.wuid+"</a>",
				"client": "<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/client/"+obj.client+"\");' style='cursor: pointer;'>"+obj.client+"</a>",
				"checkout time": obj.checkout_time,
				"cmd name": obj.cmd.name,
				"cmd args": obj.cmd.args,
				"rank": obj.rank,
				"tot": obj.totalwork,
				"state": obj.state,
				"failed": obj.failed,
				"submission time": obj.info.submittime,
				"priority": obj.info.priority
			      } );
	}
	if (! result_data.length) {
	    result_data.push({"wuid": "-", "client": "-", "checkout time": "-", "cmd name": "-", "cmd args": "-", "rank": "-", "tot": "-", "state": "-", "failed": "-"});
	}

	return result_data;
    };

    widget.dataManipulationJobs = function (data) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var result_data = [];
	for (var i=0;i<data.length;i++) {
	    var obj = data[i];
	    stm.DataStore.job[obj.id] = obj;
	    result_data.push( { "submission": obj.info.submittime,
				"job name": "<a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].jobTooltip(jQuery(this), \""+obj.id+"\")'>"+obj.info.name+"</a>", //"<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/job/"+obj.id+"\");' style='cursor: pointer;'>"+(obj.info.name || '-')+'</a>',
				"job id": obj.jid,
				"status": widget.dots(obj.tasks),
				"pipeline": obj.info.pipeline,
				"current state": obj.state
			      } );
	}
	if (! result_data.length) {
	    result_data.push({"submission": "-", "job name": "-", "job id": "-", "status": "-", "pipeline": "-", "current state": "-"});
	}

	return result_data;
    };

    /*
      Resume functions
     */
    widget.resumeClient = function (clientid) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	jQuery.ajax({
	    method: "PUT",
	    dataType: "json",
	    headers: widget.authHeader, 
	    url: RetinaConfig["awe_ip"]+"/client/"+clientid+"?resume",
	    success: function (data) {
		Retina.WidgetInstances.awe_monitor[1].updateData('clients');
		alert('client resumed');
	    }}).fail(function(xhr, error) {
		alert('failed to resume client');
	    });
    };

    widget.resumeAllClients = function () {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	jQuery.ajax({
	    method: "PUT",
	    dataType: "json",
	    headers: widget.authHeader, 
	    url: RetinaConfig["awe_ip"]+"/client?resumeall",
	    success: function (data) {
		Retina.WidgetInstances.awe_monitor[1].updateData('clients');
		alert('all clients resumed');
	    }}).fail(function(xhr, error) {
		alert('failed to resume all clients');
	    });
    };


    widget.resumeJob = function (jobid) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	jQuery.ajax({
	    method: "PUT",
	    dataType: "json",
	    headers: widget.authHeader, 
	    url: RetinaConfig["awe_ip"]+"/job/"+jobid+"?resume",
	    success: function (data) {
		Retina.WidgetInstances.awe_monitor[1].updateData('suspended');
		alert('job resumed');
	    }}).fail(function(xhr, error) {
		alert('failed to resume job');
	    });
    };

    widget.resumeAllJobs = function () {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	jQuery.ajax({
	    method: "PUT",
	    dataType: "json",
	    headers: widget.authHeader, 
	    url: RetinaConfig["awe_ip"]+"/job?resumeall",
	    success: function (data) {
		Retina.WidgetInstances.awe_monitor[1].updateData('suspended');
		alert('all jobs resumed');
	    }}).fail(function(xhr, error) {
		alert('failed to resume all jobs');
	    });
    };

    widget.xlsExport = function (job) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	// issue an XMLHttpRequest to load the empty Excel workbook from disk
	var xhr = new XMLHttpRequest();
	var method = "GET";
	var base_url = "Workbook1.xlsx";
	if ("withCredentials" in xhr) {
	    xhr.open(method, base_url, true);
	} else if (typeof XDomainRequest != "undefined") {
	    xhr = new XDomainRequest();
	    xhr.open(method, base_url);
	} else {
	    alert("your browser does not support CORS requests");
	    console.log("your browser does not support CORS requests");
	    return undefined;
	}
	
	xhr.responseType = 'arraybuffer';
	
	xhr.onload = function() {
	    var widget = Retina.WidgetInstances.awe_monitor[1];
	    
	    // the file is loaded, create a javascript object from it
	    var wb = xlsx(xhr.response);
	    
	    // create the worksheets
	    wb.worksheets[0].name = "main";
	    wb.addWorksheet({ name: "task" });
	    wb.addWorksheet({ name: "work" });
	    
	    // write the overview data
	    wb.setCell(0, 0, 0, "id");
	    wb.setCell(0, 1, 0, "queued");
	    wb.setCell(0, 2, 0, "start");
	    wb.setCell(0, 3, 0, "end");
	    wb.setCell(0, 4, 0, "resp");
	    
	    wb.setCell(0, 0, 1, job.id);
	    wb.setCell(0, 1, 1, job.queued);
	    wb.setCell(0, 2, 1, job.start);
	    wb.setCell(0, 3, 1, job.end);
	    wb.setCell(0, 4, 1, job.resp);
	    
	    // write task header
	    wb.setCell(1, 0, 0, "queued");
	    wb.setCell(1, 1, 0, "start");
	    wb.setCell(1, 2, 0, "end");
	    wb.setCell(1, 3, 0, "resp");
	    var maxInfile = 0;
	    var maxOutfile = 0;
	    for (var i in job.task_stats) {
		if (job.task_stats.hasOwnProperty(i)) {
		    if (job.task_stats[i].size_infile.length > maxInfile) {
			maxInfile = job.task_stats[i].size_infile.length;
		    }
		    if (job.task_stats[i].size_outfile.length > maxOutfile) {
			maxOutfile = job.task_stats[i].size_outfile.length;
		    }
		}
	    }
	    var currCol = 4;
	    for (var i=0; i<maxInfile; i++) {
		wb.setCell(1, currCol, 0, "size infile "+i);
		currCol++;
	    }
	    for (var i=0; i<maxOutfile; i++) {
		wb.setCell(1, currCol, 0, "size outfile "+i);
		currCol++;
	    }

	    // write task data
	    currCol = 4;
	    var currRow = 1;

	    for (var i in job.task_stats) {
		if (job.task_stats.hasOwnProperty(i)) {
		    wb.setCell(1, 0, currRow, job.task_stats[i].queued);
		    wb.setCell(1, 1, currRow, job.task_stats[i].start);
		    wb.setCell(1, 2, currRow, job.task_stats[i].end);
		    wb.setCell(1, 3, currRow, job.task_stats[i].resp);
		    var currCol = 4;
		    for (var h=0; h<job.task_stats[i].size_infile.length;h++) {
			wb.setCell(1, currCol, currRow, job.task_stats[i].size_infile[h]);
			currCol++;
		    }
		    currCol = 4 + maxInfile;
		    for (var h=0; h<job.task_stats[i].size_outfile.length;h++) {
			wb.setCell(1, currCol, currRow, job.task_stats[i].size_outfile[h]);
			currCol++;
		    }
		    currRow++;
		}
	    }
	    
	    // write work header
	    wb.setCell(2, 0, 0, "queued");
	    wb.setCell(2, 1, 0, "done");
	    wb.setCell(2, 2, 0, "resp");
	    wb.setCell(2, 3, 0, "checkout");
	    wb.setCell(2, 4, 0, "deliver");
	    wb.setCell(2, 5, 0, "clientresp");
	    wb.setCell(2, 6, 0, "time data in");
	    wb.setCell(2, 7, 0, "time data out");
	    wb.setCell(2, 8, 0, "runtime");
	    wb.setCell(2, 9, 0, "max mem usage");
	    wb.setCell(2, 10, 0, "client id");
	    wb.setCell(2, 11, 0, "size predata");
	    wb.setCell(2, 12, 0, "size infile");
	    wb.setCell(2, 13, 0, "size outfile");

	    // write work data
	    currRow = 1;
	    for (var i in job.work_stats) {
		if (job.work_stats.hasOwnProperty(i)) {
		    wb.setCell(2, 0, currRow, job.work_stats[i].queued);
		    wb.setCell(2, 1, currRow, job.work_stats[i].done);
		    wb.setCell(2, 2, currRow, job.work_stats[i].resp);
		    wb.setCell(2, 3, currRow, job.work_stats[i].checkout);
		    wb.setCell(2, 4, currRow, job.work_stats[i].deliver);
		    wb.setCell(2, 5, currRow, job.work_stats[i].clientresp);
		    wb.setCell(2, 6, currRow, job.work_stats[i].time_data_in);
		    wb.setCell(2, 7, currRow, job.work_stats[i].time_data_out);
		    wb.setCell(2, 8, currRow, job.work_stats[i].runtime);
		    wb.setCell(2, 9, currRow, job.work_stats[i].max_mem_usage);
		    wb.setCell(2, 10, currRow, job.work_stats[i].client_id);
		    wb.setCell(2, 11, currRow, job.work_stats[i].size_predata);
		    wb.setCell(2, 12, currRow, job.work_stats[i].size_infile);
		    wb.setCell(2, 13, currRow, job.work_stats[i].size_outfile);
		    currRow++;
		}
	    }

	    // open a save dialog for the user through the stm.saveAs function
	    stm.saveAs(xlsx(wb).base64, job.info.name+"_statistics.xlsx", "data:application/vnd.openxmlformats-officedocument.spreadsheetml.sheet;base64,");
	}

	xhr.send();
    };

    widget.authenticatedJSON = function (url) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	jQuery.ajax( { dataType: "json",
		       url: url,
		       headers: widget.authHeader,
		       success: function(data) {
			   stm.saveAs(JSON.stringify(data, null, 2), "data.json");
		       },
		       error: function (xhr, data) {
			   alert(JSON.parse(xhr.responseText).error[0]);
		       }
		     } );
    };

    widget.loginAction = function (action) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	if (action.action == "login" && action.result == "success") {
	    widget.authHeader = { "Authorization": "OAuth "+action.token };
	} else {
	    widget.authHeader = {};
	}
	widget.display();
    };
    
    // workflow visualization functions
    widget.stagePills = function (job) {
	var widget = this;

	var html = "<h4>this job has no tasks</h4>";
	if (job.tasks.length > 0) {
	    html = "";

	    if (job.state == "suspend") {
		html += "<div><b>job notes</b><br><pre>"+job.notes+"</pre></div>";
		html += widget.resolveError(job);
	    }

	    for (var i=0; i<job.tasks.length; i++) {
		if (job.tasks[i].state == 'completed') {
		    html += '\
<div class="pill donepill clickable" onclick="if(document.getElementById(\'stageDetails'+i+'\').style.display==\'none\'){document.getElementById(\'stageDetails'+i+'\').style.display=\'\';}else{document.getElementById(\'stageDetails'+i+'\').style.display=\'none\';};">\
  <img class="miniicon" src="Retina/images/ok.png">\
  '+job.tasks[i].cmd.description+'\
  <span style="float: right;">'+widget.prettyAWEdate(job.tasks[i].completeddate)+'</span>\
</div><div style="display: none;" id="stageDetails'+i+'">'+widget.stageDetails(job.tasks[i])+'</div>';
		} else if (job.tasks[i].state == 'in-progress') {
		    html += '\
<div class="pill runningpill">\
  <img class="miniicon" src="Retina/images/settings3.png">\
  '+job.tasks[i].cmd.description+'\
  <span style="float: right;">'+widget.prettyAWEdate(job.tasks[i].starteddate)+'</span>\
</div>';
		} else if (job.tasks[i].state == 'queued') {
		    html += '\
<div class="pill queuedpill">\
  <img class="miniicon" src="Retina/images/clock.png">\
  '+job.tasks[i].cmd.description+'\
  <span style="float: right;">(in queue)</span>\
</div>';
		} else if (job.tasks[i].state == 'error') {
			html += '\
<div class="pill errorpill clickable" onclick="if(document.getElementById(\'stageDetails'+i+'\').style.display==\'none\'){document.getElementById(\'stageDetails'+i+'\').style.display=\'\';}else{document.getElementById(\'stageDetails'+i+'\').style.display=\'none\';};">\
  <img class="miniicon" src="Retina/images/remove.png">\
  '+job.tasks[i].cmd.description+'\
  <span style="float: right;">'+widget.prettyAWEdate(job.tasks[i].createddate)+'</span>\
</div><div style="display: none;" id="stageDetails'+i+'">'+widget.stageDetails(job.tasks[i])+'</div>';
		} else if (job.tasks[i].state == 'pending') {
		    html += '\
<div class="pill pendingpill">\
  <img class="miniicon" src="Retina/images/clock.png">\
  '+job.tasks[i].cmd.description+'\
  <span style="float: right;">(not started)</span>\
</div>';
		} else if (job.tasks[i].state == 'suspend') {
			html += '\
<div class="pill errorpill clickable" onclick="if(document.getElementById(\'stageDetails'+i+'\').style.display==\'none\'){document.getElementById(\'stageDetails'+i+'\').style.display=\'\';}else{document.getElementById(\'stageDetails'+i+'\').style.display=\'none\';};">\
  <img class="miniicon" src="Retina/images/remove.png">\
  '+job.tasks[i].cmd.description+'\
  <span style="float: right;">'+widget.prettyAWEdate(job.tasks[i].createddate)+'</span>\
</div><div style="display: none;" id="stageDetails'+i+'">'+widget.stageDetails(job.tasks[i])+'</div>';
		} else {
		    console.log('unhandled state: '+job.tasks[i].state);
		}
	    }
	}

	return html;
    };

    widget.downloadHead = function (nodeid, fn) {
	var widget = this;
	jQuery.ajax({
	    method: "GET",
	    fn: fn,
	    headers: stm.authHeader,
	    url: RetinaConfig.shock_url+'/node/'+nodeid + "?download&index=size&part=1&chunksize=10240",
	    success: function (data) {
		stm.saveAs(data, this.fn);
	    }}).fail(function(xhr, error) {
		alert("could not get head of file");
		console.log(error);
	    });
    };

    widget.stageDetails = function (task) {
	var widget = this;

	var inputs = [];
	for (var i in task.inputs) {
	    if (task.inputs.hasOwnProperty(i)) {
		if (task.inputs[i].nofile || i == "mysql.tar" || i == "postgresql.tar") {
		    continue;
		}
		inputs.push(i+" ("+task.inputs[i].size.byteSize()+")" + (Retina.cgiParam('admin') ? " <button class='btn btn-mini' onclick='Retina.WidgetInstances.metagenome_pipeline[1].downloadHead(\""+task.inputs[i].node+"\", \""+i+"\");'>head 1MB</button>": ""));
	    }
	}
	inputs = inputs.join('<br>');
	var outputs = [];
	for (var i in task.outputs) {
	    if (task.outputs.hasOwnProperty(i)) {
		if (task.outputs[i].type == "update") {
		    continue;
		}
		outputs.push(i+" ("+task.outputs[i].size.byteSize()+")"+(task.outputs[i]["delete"] ? " <i>temporary</i>" : "") + (Retina.cgiParam('admin') ? " <button class='btn btn-mini' onclick='Retina.WidgetInstances.metagenome_pipeline[1].downloadHead(\""+task.outputs[i].node+"\", \""+i+"\");'>head 1MB</button>": ""));
	    }
	}
	outputs = outputs.join('<br>');
	
	var html = "<table class='table table-condensed'>";
	html += "<tr><td><b>started</b></td><td>"+widget.prettyAWEdate(task.createddate)+"</td></tr>";
	html += "<tr><td><b>completed</b></td><td>"+widget.prettyAWEdate(task.completeddate)+"</td></tr>";
	if (widget.prettyAWEdate(task.completeddate) == "-") {
	    html += "<tr><td><b>duration</b></td><td>-</td></tr>";
	} else {
	    html += "<tr><td><b>duration</b></td><td>"+widget.timePassed(Date.parse(task.createddate), Date.parse(task.completeddate))+"</td></tr>";
	}
	html += "<tr><td><b>inputs</b></td><td>"+inputs+"</td></tr>";
	html += "<tr><td><b>outputs</b></td><td>"+outputs+"</td></tr>";
	html += "</table>";
	
	return html;
    };

    widget.prettyAWEdate = function (date) {
	if (date == "0001-01-01T00:00:00Z") {
	    return "-";
	}
	var pdate = new Date(Date.parse(date)).toLocaleString();
	return pdate;
    };

     widget.timePassed = function (start, end) {
	// time since submission
	var time_passed = end - start;
	var day = parseInt(time_passed / (1000 * 60 * 60 * 24));
	time_passed = time_passed - (day * 1000 * 60 * 60 * 24);
	var hour = parseInt(time_passed / (1000 * 60 * 60));
	time_passed = time_passed - (hour * 1000 * 60 * 60);
	var minute = parseInt(time_passed / (1000 * 60));
	var some_time = ((day > 0) ? day+" days " : "") + ((hour > 0) ? hour+" hours " : "") + minute+" minutes";
	return some_time;
    };

    widget.resolveError = function (job) {
	var widget = this;

	var html = "<div id='errorJob"+job.id+"'></div>";

	jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/work/"+job.lastfailed,
			   headers: widget.authHeader,
			   success: function (data) {
			       var target = document.getElementById('errorJob'+job.id);
			       target.innerHTML += "<div><b>workunit notes</b><br><pre>"+(data.data.notes || "-")+"</pre></div>";//<br><pre>"+JSON.stringify(data, null, 2)+"</pre>";
			   }
		     });

	jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/work/"+job.lastfailed+"?report=stdout",
			   headers: widget.authHeader,
			   success: function (data) {
			       var target = document.getElementById('errorJob'+job.id);
			       target.innerHTML += "<div><b>workunit stdout</b><br><pre>"+data.data+"</pre></div>";
			   }
		     });

	jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/work/"+job.lastfailed+"?report=stderr",
			   headers: widget.authHeader,
			   success: function (data) {
			       var target = document.getElementById('errorJob'+job.id);
			       target.innerHTML += "<div><b>workunit stderr</b><br><pre>"+data.data+"</pre></div>";
			   }
		     });

	jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/work/"+job.lastfailed+"?report=worknotes",
			   headers: widget.authHeader,
			   success: function (data) {
			       var target = document.getElementById('errorJob'+job.id);
			       target.innerHTML += "<div><b>workunit worknotes</b><br><pre>"+data.data+"</pre></div>";
			   }
		     });
	
	return html;
    };
    
})();
