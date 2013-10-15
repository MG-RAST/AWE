(function () {
    widget = Retina.Widget.extend({
        about: {
                title: "AWE Task Status Monitor",
                name: "awe_monitor",
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
		widget.tables[views[i]] = Retina.Renderer.create("table", { target: target_space, data: {}, filter_autodetect: true, sort_autodetect: true }); 
	    }
	    
	    widget.update_data(views[i]);
	}
    };

    widget.update_data = function (which) {
	var return_data = {};

	switch (which) {
	case "overview":	    
	    jQuery.getJSON("http://"+RetinaConfig["awe_ip"]+"/queue", function(data) {
		var result = data.data;
		var rows = result.split("\n");
		
		return_data = { "total jobs": { "all": rows[1].match(/\d+/)[0],
						"in-progress": rows[2].match(/\d+/)[0],
						"suspended": rows[3].match(/\d+/)[0] },
				"total tasks": { "all": rows[4].match(/\d+/)[0],
						 "queuing": rows[5].match(/\d+/)[0],
						 "pending": rows[6].match(/\d+/)[0],
						 "completed": rows[7].match(/\d+/)[0],
						 "suspended": rows[8].match(/\d+/)[0],
						 "failed and skipped": rows[9].match(/\d+/)[0] },
				"total workunits": { "all": rows[10].match(/\d+/)[0],
						     "queueing": rows[11].match(/\d+/)[0],
						     "checkout": rows[12].match(/\d+/)[0],
						     "suspended": rows[13].match(/\d+/)[0] },
				"total clients": { "all": rows[14].match(/\d+/)[0],
						   "busy": rows[15].match(/\d+/)[0],
						   "idle": rows[16].match(/\d+/)[0],
						   "suspended": rows[17].match(/\d+/)[0] } };
		
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
	    });
	    return;

	    break;
	case "graphical":
	    jQuery.getJSON("http://"+RetinaConfig["awe_ip"]+"/job", function (data) {
		var result_data = [];
		if (data.data != null) {
		    var data2 = [];
		    for (var i=0;i<data.data.length;i++) {
			if ((data.data[i].state == 'in-progress') || (data.data[i].state == 'suspend') || (data.data[i].state == 'submitted')) {

			    data2.push(data.data[i]);
			}
		    }
		    data2 = data2.sort(widget.tasksort);
		    for (h=0;h<data2.length;h++) {
			var obj = data2[h];
			result_data.push( [ obj.info.submittime,
					    "<a href='http://"+RetinaConfig["awe_ip"]+"/job/"+obj.id+"' target=_blank>"+(obj.info.name || '-')+' ('+obj.jid+")</a>",
					    widget.dots(obj.tasks),
					    obj.info.pipeline,
					    obj.state
					  ] );
		    }
		}
		if (! result_data.length) {
		    result_data.push(['-','-','-','-','-']);
		}
		return_data = { header: [ "submission", "job", "status", "pipeline", "current state" ],
				data: result_data };
		Retina.WidgetInstances.awe_monitor[1].tables["graphical"].settings.rows_per_page = 20;
		Retina.WidgetInstances.awe_monitor[1].tables["graphical"].settings.minwidths = [1,300,1, 1, 1];
		Retina.WidgetInstances.awe_monitor[1].tables["graphical"].settings.data = return_data;
		Retina.WidgetInstances.awe_monitor[1].tables["graphical"].render();
		Retina.WidgetInstances.awe_monitor[1].check_update();
	    });

	    break;
	case "active":
	    jQuery.getJSON("http://"+RetinaConfig["awe_ip"]+"/job?active", function (data) {
		var result_data = [];
		if (data.data != null) {
		    for (h=0;h<data.data.length;h++) {
			var obj = data.data[h];
			result_data.push( [ obj.info.submittime,
					    "<a href='http://"+RetinaConfig["awe_ip"]+"/job/"+obj.id+"' target=_blank>"+obj.jid+"</a>",
					    obj.info.name,
					    obj.info.user,
					    obj.info.project,
					    obj.info.pipeline,
					    obj.info.clientgroups,
					    obj.tasks.length - obj.remaintasks || "0",
					    obj.tasks.length,
					    obj.state,
                                            obj.updatetime
					  ] );
		    }
		}
		if (! result_data.length) {
		    result_data.push(['-','-','-','-','-','-','-','-','-','-','-']);
		}
		return_data = { header: [ "created",
					  "jid",
					  "name",
					  "user",
					  "project",
					  "pipeline",
					  "group",
					  "t-complete",
					  "t-total",
					  "state", 
                                          "updated"],
				data: result_data };
		Retina.WidgetInstances.awe_monitor[1].tables["active"].settings.minwidths = [1,1,65,1,75,85,90,10,10,75,75];
		Retina.WidgetInstances.awe_monitor[1].tables["active"].settings.data = return_data;
		Retina.WidgetInstances.awe_monitor[1].tables["active"].render();
		Retina.WidgetInstances.awe_monitor[1].check_update();
	    });

	    break;
	case "suspended":
	    jQuery.getJSON("http://"+RetinaConfig["awe_ip"]+"/job?suspend", function (data) {
		var result_data = [];
		if (data.data != null) {
		    for (h=0;h<data.data.length;h++) {
			var obj = data.data[h];
			result_data.push( [ obj.info.submittime,
                                            "<a href='http://"+RetinaConfig["awe_ip"]+"/job/"+obj.id+"' target=_blank>"+obj.jid+"</a>",
					    obj.info.name,
					    obj.info.user,
					    obj.info.project,
					    obj.info.pipeline,
					    obj.info.clientgroups,
					    obj.tasks.length - obj.remaintasks || "0",
					    obj.tasks.length,
					    obj.state,
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
					  "t-complete",
					  "t-total",
					  "state",
                                          "updated"],
				data: result_data };

		Retina.WidgetInstances.awe_monitor[1].tables["suspended"].settings.minwidths = [1,1,65,1,75,85,90,10,10,75,75];
		Retina.WidgetInstances.awe_monitor[1].tables["suspended"].settings.data = return_data;
		Retina.WidgetInstances.awe_monitor[1].tables["suspended"].render();
		Retina.WidgetInstances.awe_monitor[1].check_update();
	    });

	    break;
	case "queuing_workunit":
	    jQuery.getJSON("http://"+RetinaConfig["awe_ip"]+"/work?query&state=queued", function (data) {
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
					    obj.failed || "0"
					  ] );
		    }
		}
		if (! result_data.length) {
		    result_data.push(['-','-','-','-','-','-','-','-']);
		}
		return_data = { header: [ "wuid",
					  "submission time",
					  "cmd name",
					  "cmd args",
					  "rank",
					  "t-work",
					  "state",
					  "failed" ],
				data: result_data };

		Retina.WidgetInstances.awe_monitor[1].tables["queuing_workunit"].settings.minwidths = [1,1,1,1,65,75,75,75];
		Retina.WidgetInstances.awe_monitor[1].tables["queuing_workunit"].settings.data = return_data;
		Retina.WidgetInstances.awe_monitor[1].tables["queuing_workunit"].render();
		Retina.WidgetInstances.awe_monitor[1].check_update();
	    });

	    break;
	case "completed":
	    var options = document.getElementById('completed_options');
	    if (! options) {
		options = document.createElement('div');
		options.setAttribute('id', 'completed_options');
		document.getElementById('completed').insertBefore(options, document.getElementById('completed').firstChild);
		options.innerHTML = "<span style='position: relative; bottom: 4px;'>retrieve the last </span><input id='num_recent' value='10' class='span1' style='bottom: 1px; position:relative;'></id><span style='position: relative; bottom: 4px;'> entries </span> <input type='button' class='btn btn-mini' value='update' onclick='Retina.WidgetInstances.awe_monitor[1].update_data(\"completed\");' style='bottom: 5px; position:relative;'>";
	    }
	    var num_recent = document.getElementById('num_recent').value;
	    if (isNaN(num_recent)) {
		num_recent = "10";
		document.getElementById('num_recent').value = "10";
		alert('You may only enter numbers, defaulting to 10.');
		num_recent = "&recent=10";
	    } else {
		num_recent = parseInt(num_recent);
		if (num_recent > 0) {
		    num_recent = "&recent=" + num_recent;
		} else {
		    num_recent = "";
		}
	    }
	    jQuery.getJSON("http://"+RetinaConfig["awe_ip"]+"/job?query&state=completed"+num_recent, function (data) {
		var result_data = [];
		if (data.data != null) {
		    for (h=0;h<data.data.length;h++) {
			var obj = data.data[h];
			result_data.push( [ obj.info.submittime,
					    "<a href='http://"+RetinaConfig["awe_ip"]+"/job/"+obj.id+"' target=_blank>"+obj.jid+"</a>",
					    obj.info.name,
					    obj.info.user,
					    obj.info.project,
					    obj.info.pipeline,
					    obj.info.clientgroups,
					    obj.tasks.length - obj.remaintasks || "0",
					    obj.tasks.length,
					    obj.state,
                                            obj.updatetime
					  ] );
		    }
		}
		if (! result_data.length) {
		    result_data.push(['-','-','-','-','-','-','-','-','-','-','-']);
		}
		return_data = { header: [ "created",
					  "jid",
					  "name",
					  "user",
					  "project",
					  "pipeline",
					  "group",
					  "t-complete",
					  "t-total",
					  "state", 
                                          "finished"],
				data: result_data };
		Retina.WidgetInstances.awe_monitor[1].tables["completed"].settings.minwidths = [1,1,65,1,75,85,90,10,10,75,75];
		Retina.WidgetInstances.awe_monitor[1].tables["completed"].settings.tdata ? delete Retina.WidgetInstances.awe_monitor[1].tables["completed"].settings.tdata : "";
		Retina.WidgetInstances.awe_monitor[1].tables["completed"].settings.data = return_data;
		Retina.WidgetInstances.awe_monitor[1].tables["completed"].render();
		Retina.WidgetInstances.awe_monitor[1].check_update();
	    });

	    break;
	case "checkout_workunit":
	    jQuery.getJSON("http://"+RetinaConfig["awe_ip"]+"/work?query&state=checkout", function (data) {
		var result_data = [];
		if (data.data != null) {
		    for (h=0;h<data.data.length;h++) {
			var obj = data.data[h];
			result_data.push( [ "<a href='http://"+RetinaConfig["awe_ip"]+"/work/"+obj.wuid+"' target=_blank>"+obj.wuid+"</a>",
                                            "<a href='http://"+RetinaConfig["awe_ip"]+"/client/"+obj.client+"' target=_blank>"+obj.client+"</a>",
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
					  "t-work",
					  "state",
					  "failed" ],
				data: result_data };

		Retina.WidgetInstances.awe_monitor[1].tables["checkout_workunit"].settings.minwidths = [1,1,1,1,1,50,50,50,50];
		Retina.WidgetInstances.awe_monitor[1].tables["checkout_workunit"].settings.data = return_data;
		Retina.WidgetInstances.awe_monitor[1].tables["checkout_workunit"].render();
		Retina.WidgetInstances.awe_monitor[1].check_update();
	    });

	    break;
	case "clients":
	    jQuery.getJSON("http://"+RetinaConfig["awe_ip"]+"/client", function (data) {
		var result_data = [];
		if (data.data == null) {
		    result_data = [ ['-','-','-','-','-','-','-','-','-','-','-','-','-','-','-'] ];
		} else {
		    for (h=0;h<data.data.length;h++) {
			var obj = data.data[h];
			result_data.push( ["<a href='http://"+RetinaConfig["awe_ip"]+"/client/"+obj.id+"' target=_blank>"+obj.name+"</a>",
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
					    obj.skip_work.join(", ")]);
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

		Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.minwidths = [70,73,50,70,73,1,115,83,70,90,75,57,67,68,90];
		Retina.WidgetInstances.awe_monitor[1].tables["clients"].settings.data = return_data;
		Retina.WidgetInstances.awe_monitor[1].tables["clients"].render();
		Retina.WidgetInstances.awe_monitor[1].check_update();
	    }).error(function(){
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
	Retina.WidgetInstances.awe_monitor[1].updated += 100 / 6;
	if (parseInt(Retina.WidgetInstances.awe_monitor[1].updated) == 100) {
	    document.getElementById('refresh').innerHTML = '<button class="btn" onclick="Retina.WidgetInstances.awe_monitor[1].display();">refresh</button>';
	} else {
	    document.getElementById('pbar') ? document.getElementById('pbar').setAttribute('style', "width: "+Retina.WidgetInstances.awe_monitor[1].updated+"%;") : "";
	}
    };

    widget.dots = function (stages) {
	var dots = '<span>';
	if (stages.length > 0) {
	    for (var i=0;i<stages.length;i++) {
		if (stages[i].state == 'completed') {
		    dots += '<span style="color: green;font-size: 19px; cursor: default;" title="completed: '+stages[i].cmd.description+'">&#9679;</span>';
		} else if (stages[i].state == 'in-progress') {
		    dots += '<span style="color: blue;font-size: 19px; cursor: default;" title="in-progress: '+stages[i].cmd.description+'">&#9679;</span>';
		} else if (stages[i].state == 'queued') {
		    dots += '<span style="color: orange;font-size: 19px; cursor: default;" title="queued: '+stages[i].cmd.description+'">&#9679;</span>';
		} else if (stages[i].state == 'error') {
		    dots += '<span style="color: red;font-size: 19px; cursor: default;" title="error: '+stages[i].cmd.description+'">&#9679;</span>';
		} else if (stages[i].state == 'init') {
		    dots += '<span style="color: gray;font-size: 19px; cursor: default;" title="init: '+stages[i].cmd.description+'">&#9679;</span>';
		}
	    }
	}
			  
	dots += "</span>";

	return dots;
    };

    widget.tasksort = function (a, b) {
	var order = { "suspend": 0, "submitted": 1, "in-progress": 2 };
	if (order[a.state] > order[b.state]) {
	    return -1;
	} else if (order[a.state] < order[b.state]) {
	    return 1;
	} else {
	    return a.info.submittime.localeCompare(b.info.submittime);
	}
    };

})();
