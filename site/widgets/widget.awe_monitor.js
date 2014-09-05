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
	    return [ Retina.load_renderer("table") ];
    };

    widget.tables = [];
        
    widget.display = function (wparams) {
	// initialize
        widget = this;
	var index = widget.index;

	widget.target = widget.target || wparams.target;
	widget.target.innerHTML = '\
<div id="refresh" style="position: absolute; top: 64px; left: 17px; z-index: 100;">\
    </div>\
    <div id="overview" class="sidebar"></div>\
    <div class="mainview">\
      <ul class="nav nav-tabs">\
	<li class="active">\
	  <a data-toggle="tab" href="#graphical">overview</a>\
	</li>\
	<li>\
	  <a data-toggle="tab" href="#active">active jobs</a>\
	</li>\
	<li class="">\
	  <a data-toggle="tab" href="#suspended">suspended jobs</a>\
	</li>\
	<li class="">\
	  <a data-toggle="tab" href="#completed">completed jobs</a>\
	</li>\
	<li class="">\
	  <a data-toggle="tab" href="#queuing_workunit">queued workunits</a>\
	</li>\
	<li class="">\
	  <a data-toggle="tab" href="#checkout_workunit">checked-out workunits</a>\
	</li>\
	<li class="">\
	  <a data-toggle="tab" href="#clients">clients</a>\
	</li>\
      </ul>\
      <div class="tab-content">\
	<div id="graphical" class="tab-pane active">\
	</div>\
	<div id="active" class="tab-pane">\
	</div>\
	<div id="suspended" class="tab-pane">\
	</div>\
	<div id="completed" class="tab-pane">\
	</div>\
	<div id="queuing_workunit" class="tab-pane">\
	</div>\
	<div id="checkout_workunit" class="tab-pane">\
	</div>\
	<div id="clients" class="tab-pane">\
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

	var views = [ "overview",
		      "graphical",
		      "active",
		      "suspended",
		      "completed",
		      "queuing_workunit",
		      "checkout_workunit",
		      "clients" ];

	for (i=0;i<views.length;i++) {
	    var view = document.getElementById(views[i]);
	    view.innerHTML = "";
	   
	    if (views[i] != "overview") {
		var target_space = document.createElement('div');
		target_space.innerHTML = "";
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
	case "graphical":
	    var gt = Retina.WidgetInstances.awe_monitor[1].tables["graphical"];
	    gt.settings.headers = widget.authHeader;
	    gt.settings.synchronous = false;
	    gt.settings.query_type = 'prefix';
	    gt.settings.data_manipulation = Retina.WidgetInstances.awe_monitor[1].dataManipulationGraphical,
	    gt.settings.navigation_url = RetinaConfig["awe_ip"]+"/job?query";
	    gt.settings.rows_per_page = 20;
	    gt.settings.minwidths = [1,300,1, 95, 125];
	    gt.settings.disable_sort = { 2: 1 };
	    gt.settings.filter = { 1: { type: "text" },
				   3: { type: "text" },
				   4: { type: "text" } };
	    gt.settings.asynch_column_mapping = { "submission": "info.submittime",
						  "job": "info.name",
						  "pipeline": "info.pipeline",
						  "current state": "state" };
	    gt.settings.filter_autodetect = false;
	    gt.settings.sort_autodetect = false;
	    gt.settings.data = { data: [], header: [ "submission", "job", "status", "pipeline", "current state" ] };
	    gt.render();
	    gt.update({}, gt.index);

	    break;
	case "active":
	    var at = Retina.WidgetInstances.awe_monitor[1].tables["active"];
	    at.settings.headers = widget.authHeader;
	    at.settings.synchronous = false;
	    at.settings.query_type = 'prefix';
	    at.settings.data_manipulation = Retina.WidgetInstances.awe_monitor[1].dataManipulationActive,
	    at.settings.navigation_url = RetinaConfig["awe_ip"]+"/job?active";
	    at.settings.rows_per_page = 20;
	    at.settings.minwidths = [1,1,65,1,85,85,90,40,40,75,75,83];
	    at.settings.disable_sort = { };
	    at.settings.filter = { };
	    at.settings.asynch_column_mapping = { "created": "info.submittime",
						  "jid": "jid",
						  "name": "info.name",
						  "user": "info.user",
						  "project": "info.project",
						  "pipeline": "info.pipeline",
						  "group": "info.clientgroups",
						  "tot": "tasks.length",
						  "state": "state",
						  "updated": "updatetime",
						  "priority": "info.priority" };
	    at.settings.filter_autodetect = false;
	    at.settings.sort_autodetect = false;
	    at.settings.data = { data: [], header: [ "created", "jid", "name", "user", "project", "pipeline", "group", "ok", "tot", "state", "updated", "priority" ] };
	    at.render();
	    at.update({}, at.index);

	    break;
	case "suspended":
	    jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/job?suspend&limit=1000",
			   headers: widget.authHeader,
			   error: function () {
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   },
			   success: function (data) {
			       var widget = Retina.WidgetInstances.awe_monitor[1];
			       var result_data = [];
			       if (data.data != null) {
				   for (h=0;h<data.data.length;h++) {
				       var obj = data.data[h];
				       result_data.push( [ obj.info.submittime,
							   "<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/job/"+obj.id+"\");' style='cursor: pointer;'>"+obj.jid+"</a>",
							   obj.info.name,
							   obj.info.user,
							   obj.info.project,
							   obj.info.pipeline,
							   obj.info.clientgroups,
							   obj.tasks.length - obj.remaintasks || "0",
							   obj.tasks.length,
							   "<a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].clientTooltip(jQuery(this), \""+obj.lastfailed+"\")'>"+obj.lastfailed+"</a>",
							   obj.updatetime
							 ] );
				   }
			       }
			       if (! result_data.length) {
				   result_data.push(['-','-','-','-','-','-','-','-','-','-','-']);
			       }
			       return_data = { header: [ "submitted",
							 "jid",
							 "name",
							 "user",
							 "project",
							 "pipeline",
							 "group",
							 "ok",
							 "tot",
							 "state",
							 "updated"],
					       data: result_data };

			       Retina.WidgetInstances.awe_monitor[1].tables["suspended"].settings.minwidths = [1,55,75,1,75,85,90,55,55,75,75];
			       Retina.WidgetInstances.awe_monitor[1].tables["suspended"].settings.invisible_columns = { 10: true };
			       Retina.WidgetInstances.awe_monitor[1].tables["suspended"].settings.data = return_data;
			       Retina.WidgetInstances.awe_monitor[1].tables["suspended"].render();
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   }
	    });

	    break;
	case "queuing_workunit":
	    jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/work?query&state=queued",
			   headers: widget.authHeader,
			   error: function () {
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   },
			   success: function (data) {
			       var widget = Retina.WidgetInstances.awe_monitor[1];
			       var result_data = [];
			       if (data.data != null) {
				   for (h=0;h<data.data.length;h++) {
				       var obj = data.data[h];
				       result_data.push( [ obj.wuid,
							   obj.info.submittime,
							   obj.cmd.name,
							   obj.cmd.args,
							   obj.rank || "0",
							   obj.totalwork || "0",
							   obj.state,
							   obj.failed || "0",
							   obj.info.priority
							 ] );
				   }
			       }
			       if (! result_data.length) {
				   result_data.push(['-','-','-','-','-','-','-','-','-']);
			       }
			       return_data = { header: [ "wuid",
							 "submission time",
							 "cmd name",
							 "cmd args",
							 "rank",
							 "tot",
							 "state",
							 "failed",
							 "priority" ],
					       data: result_data };

			       Retina.WidgetInstances.awe_monitor[1].tables["queuing_workunit"].settings.minwidths = [1,1,1,1,65,78,75,75,83];
			       Retina.WidgetInstances.awe_monitor[1].tables["queuing_workunit"].settings.data = return_data;
			       Retina.WidgetInstances.awe_monitor[1].tables["queuing_workunit"].render();
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   }
			 });
	    break;
	case "completed":
	    var ct = Retina.WidgetInstances.awe_monitor[1].tables["completed"];
	    ct.settings.headers = widget.authHeader;
	    ct.settings.synchronous = false;
	    ct.settings.query_type = 'prefix';
	    ct.settings.data_manipulation = Retina.WidgetInstances.awe_monitor[1].dataManipulationCompleted,
	    ct.settings.navigation_url = RetinaConfig["awe_ip"]+"/job?query&state=completed";
	    ct.settings.rows_per_page = 20;
	    ct.settings.minwidths = [1,51,75,64,83,85,90,107,75,75,75];
	    ct.settings.asynch_column_mapping = { "created": "info.submittime",
						  "jid": "jid",
						  "name": "info.name",
						  "user": "info.user",
						  "project": "info.project",
						  "pipeline": "info.pipeline",
						  "group": "info.clientgroups",
						  "state": "state", 
						  "finished": "updatetime" };
	    ct.settings.filter = { 0: { type: "text" },
				   1: { type: "text" },
				   2: { type: "text" },
				   3: { type: "text" },
				   4: { type: "text" },
				   5: { type: "text" },
				   6: { type: "text" },
				   9: { type: "text" },
				   10: { type: "text" } };
	    ct.settings.disable_sort = { 7: 1, 8: 1 };
	    ct.settings.filter_autodetect = false;
	    ct.settings.sort_autodetect = false;
	    ct.settings.data = { data: [], header: [ "created",
						     "jid",
						     "name",
						     "user",
						     "project",
						     "pipeline",
						     "group",
						     "ok",
						     "tot",
						     "state", 
						     "finished" ] };
	    ct.render();
	    ct.update({}, ct.index);

	    break;
	case "checkout_workunit":
	    jQuery.ajax( { dataType: "json",
			   url: RetinaConfig["awe_ip"]+"/work?query&state=checkout",
			   headers: widget.authHeader,
			   error: function () {
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   },
			   success: function (data) {
			       var widget = Retina.WidgetInstances.awe_monitor[1];
			       var result_data = [];
			       if (data.data != null) {
				   for (h=0;h<data.data.length;h++) {
				       var obj = data.data[h];
				       result_data.push( [ "<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/work/"+obj.wuid+"\");' style='cursor: pointer;'>"+obj.wuid+"</a>",
							   "<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/client/"+obj.client+"\");' style='cursor: pointer;'>"+obj.client+"</a>",
							   obj.checkout_time,
							   obj.cmd.name,
							   obj.cmd.args,
							   obj.rank || "0",
							   obj.totalwork || "0",
							   obj.state,
							   obj.failed || "0"
							 ] );
				   }
			       }
			       if (! result_data.length) {
				   result_data.push(['-','-','-','-','-','-','-','-','-']);
			       }
			       return_data = { header: [ "wuid",
							 "client",
							 "checkout time",
							 "cmd name",
							 "cmd args",
							 "rank",
							 "tot",
							 "state",
							 "failed" ],
					       data: result_data };

			       Retina.WidgetInstances.awe_monitor[1].tables["checkout_workunit"].settings.minwidths = [1,1,1,1,1,65,65,75,75];
			       Retina.WidgetInstances.awe_monitor[1].tables["checkout_workunit"].settings.data = return_data;
			       Retina.WidgetInstances.awe_monitor[1].tables["checkout_workunit"].render();
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   }
	    });

	    break;
	case "clients":
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
					   skipwork.push("<a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].clientTooltip(jQuery(this), \""+obj.skip_work[j]+"\")'>"+obj.skip_work[j]+"</a>");
				       }
				       result_data.push( [ "<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/client/"+obj.id+"\");' style='cursor: pointer;'>"+obj.name+"</a>",
							  obj.group,
							  obj.user || "-",
							  obj.host,
							  obj.cores || "0",
							  obj.apps.join(", "),
							  obj.regtime,
							  obj.serve_time,
							  obj.proxy || "-",
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
							 "user",
							 "host",
							 "cores",
							 "apps",
							 "register time",
							 "up-time",
							 "proxy",
							 "subclients",
							 "status",
							 "c/o",
							 "done",
							 "failed",
							 "failed wuid"],
					       data: result_data };

			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.rows_per_page = 100;
			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.minwidths = [70,73,50,70,73,1,115,83,70,90,75,57,67,68,90];
			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.data = return_data;
			       Retina.WidgetInstances.awe_monitor[1].tables["clients"].render();
			       Retina.WidgetInstances.awe_monitor[1].check_update();
			   }
	    }).error(function(){
		var widget = Retina.WidgetInstances.awe_monitor[1];
		var result_data = [ ['-','-','-','-','-','-','-','-','-','-','-','-','-'] ];
		return_data = { header: [ "id",
					  "name",
					  "group",
					  "user",
					  "host",
					  "cores",
					  "apps",
					  "register time",
					  "up-time",
					  "status",
					  "c/o",
					  "done",
					  "failed" ],
				data: result_data };
		Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.minwidths = [1,1,73,73,70,73,1,115,83,75,57,67,68];
		Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.data = return_data;
		Retina.WidgetInstances.awe_monitor[1].tables["clients"].render();
		Retina.WidgetInstances.awe_monitor[1].check_update();
	    });

	    break;
	default:
	    return null;
	}
    };
    
    widget.check_update = function () {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	Retina.WidgetInstances.awe_monitor[1].updated += 100 / 5;
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

    widget.tooltip = function (obj, id) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	obj.popover('destroy');
	obj.popover({content: "<button class='close' style='position: relative; bottom: 8px; left: 8px;' type='button' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>×</button><a style='cursor: pointer;' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/job/"+id+"\");'>job details</a><br><a style='cursor: pointer;' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/job/"+id+"?perf\");'>job stats</a><br><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].jobDetails(&#39;"+id+"&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>excel</a>",html:true,placement:"top"});
	obj.popover('show');
    }

    widget.clientTooltip = function (obj, id) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	obj.popover('destroy');
	obj.popover({content: "<button class='close' style='position: relative; bottom: 8px; left: 8px;' type='button' onclick='this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>×</button><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].workunitDetails(&#39;"+id+"&#39;,&#39;stderr&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>error</a><br><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].workunitDetails(&#39;"+id+"&#39;,&#39;stdout&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>output</a><br><a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].workunitDetails(&#39;"+id+"&#39;,&#39;worknotes&#39;);this.parentNode.parentNode.parentNode.removeChild(this.parentNode.parentNode);'>worknotes</a>",html:true,placement:"top"});
	obj.popover('show');
    }

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

    widget.jobDetails = function (jobid) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	jQuery.ajax({ 
	    dataType: "json",
	    headers: widget.authHeader, 
	    url: RetinaConfig["awe_ip"]+"/job/"+jobid,
	    success: function (data) {
		var widget = Retina.WidgetInstances.awe_monitor[1];
		var job = data.data;
		jQuery.getJSON(RetinaConfig["awe_ip"]+"/job/"+jobid+"?perf", function (data) {
		    var widget = Retina.WidgetInstances.awe_monitor[1];
		    job.queued = data.data.queued;
		    job.start = data.data.start;
		    job.end = data.data.end;
		    job.resp = data.data.resp;
		    job.task_stats = data.data.task_stats;
		    job.work_stats = data.data.work_stats;
		    Retina.WidgetInstances.awe_monitor[1].xlsExport(job);
		}).fail(function() {
		    var widget = Retina.WidgetInstances.awe_monitor[1];
		    alert('no job statistics available');
		});
	    }
	});
    };

    widget.dataManipulationGraphical = function (data) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var result_data = [];
	for (var i=0;i<data.length;i++) {
	    var obj = data[i];
	    result_data.push( { "submission": obj.info.submittime,
				"job": "<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/job/"+obj.id+"\");' style='cursor: pointer;'>"+(obj.info.name || '-')+' ('+obj.jid+")</a>",
				"status": widget.dots(obj.tasks),
				"pipeline": obj.info.pipeline,
				"current state": obj.state
			      } );
	}
	if (! result_data.length) {
	    result_data.push({"submission": "-", "job": "-", "status": "-", "pipeline": "-", "current state": "-"});
	}

	return result_data;
    };

    widget.dataManipulationCompleted = function (data) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var result_data = [];
	for (var i=0;i<data.length;i++) {
	    var obj = data[i];
	    result_data.push( { "created": obj.info.submittime,
				"jid": "<a style='cursor: pointer;' onclick='Retina.WidgetInstances.awe_monitor[1].tooltip(jQuery(this), \""+obj.id+"\")'>"+obj.jid+"</a>",
				"name": obj.info.name,
				"user": obj.info.user,
				"project": obj.info.project,
				"pipeline": obj.info.pipeline,
				"group": obj.info.clientgroups,
				"ok": obj.tasks.length - obj.remaintasks || "0",
				"tot": obj.tasks.length,
				"state": obj.state, 
				"finished": obj.updatetime  
			      } );
	}
	if (! result_data.length) {
	    result_data.push({ "created": "-",
			       "jid": "-",
			       "name": "-",
			       "user": "-",
			       "project": "-",
			       "pipeline": "-",
			       "group": "-",
			       "ok": "-",
			       "tot": "-",
			       "state": "-", 
			       "finished": "-" });
	}

	return result_data;
    };

    widget.dataManipulationActive = function (data) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var result_data = [];
	for (var i=0;i<data.length;i++) {
	    var obj = data[i];
	    result_data.push( { "created": obj.info.submittime,
				"jid": "<a onclick='Retina.WidgetInstances.awe_monitor[1].authenticatedJSON(\""+RetinaConfig["awe_ip"]+"/job/"+obj.id+"\");' style='cursor: pointer;'>"+obj.jid+"</a>",
				"name": obj.info.name,
				"user": obj.info.user,
				"project": obj.info.project,
				"pipeline": obj.info.pipeline,
				"group": obj.info.clientgroups,
				"ok": obj.tasks.length - obj.remaintasks || "0",
				"tot": obj.tasks.length,
				"state": obj.state,
				"updated": obj.updatetime,
				"priority": obj.info.priority
			      } );
	}
	if (! result_data.length) {
	    result_data.push({ "created": "-",
			       "jid": "-",
			       "name": "-",
			       "user": "-",
			       "project": "-",
			       "pipeline": "-",
			       "group": "-",
			       "ok": "-",
			       "tot": "-",
			       "state": "-", 
			       "updated": "-",
			       "priority": "-" });
	}
	return result_data;
    };

    widget.dataManipulationQueued = function (data) {
	var widget = Retina.WidgetInstances.awe_monitor[1];
	var result_data = [];
	for (var i=0;i<data.length;i++) {
	    var obj = data[i];
	    result_data.push( { "wuid": obj.wuid,
				"submission time": obj.info.submittime,
				"cmd name": obj.cmd.name,
				"cmd args": obj.cmd.args,
				"rank": obj.rank || "0",
				"tot": obj.totalwork || "0",
				"state": obj.state,
				"failed": obj.failed || "0",
				"priority": obj.info.priority
			      } );
	}
	if (! result_data.length) {
	    result_data.push({ "wuid": "-",
			       "submission time": "-",
			       "cmd name": "-",
			       "cmd args": "-",
			       "rank": "-",
			       "tot": "-",
			       "state": "-",
			       "failed": "-",
			       "priority": "-" });
	}
	return result_data;
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
			   var w = window.open();
			   w.document.write("<h3>"+this.url+"</h3>");
			   w.document.write("<pre>"+JSON.stringify(data, null, 2)+"</pre>");
			   w.document.close();
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
    
})();
