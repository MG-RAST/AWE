(function () {
    var widget = Retina.Widget.extend({
        about: {
            title: "AWE Panel",
            name: "awe_panel",
            author: "Tobias Paczian",
            requires: [ "rgbcolor.js" ]
        }
    });

    widget.history = [ [ Date.now(), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0 ] ];
    widget.maxEntries = RetinaConfig.maxHistoryEntries || 100;
    
    widget.setup = function () {
	return [ Retina.load_renderer("plot") ];
    };
    
    widget.display = function (wparams) {
	widget = Retina.WidgetInstances.awe_panel[1];
	
	widget.target = widget.target || wparams.target;
	
	if (! RetinaConfig.authentication || widget.loggedIn) {
	    widget.target.innerHTML = "<div id='awe_details' style='margin-left: 50px; margin-top: 50px;'></div>";
	    widget.getAWEDetails();
	} else {
	    widget.target.innerHTML = "<p>authorization required</p>";
	}
    };
    
    widget.showAWEDetails = function () {
	var widget = Retina.WidgetInstances.awe_panel[1];
	
	var target = document.getElementById('awe_details');
	
	var data = widget.aweClientData;

	var coreGroups = {};
	var stats = { stati: {} };
	for (var i=0; i<data.length; i++) {
	    if (! stats.stati.hasOwnProperty(data[i].Status)) {
		stats.stati[data[i].Status] = 0;
	    }
	    stats.stati[data[i].Status]++;
	    if (! coreGroups.hasOwnProperty(data[i].cores)) {
		coreGroups[data[i].cores] = [];
	    }
	    coreGroups[data[i].cores].push(data[i]);
	}
	for (var i in coreGroups) {
	    if (coreGroups.hasOwnProperty(i)) {
		coreGroups[i].sort(Retina.propSort("id"));
	    }
	}
	var clientData = { all: data.length };
	for (var i in stats.stati) {
	    if (stats.stati.hasOwnProperty(i)) {
		clientData[i] = stats.stati[i];
	    }
	}
	
	// client counts
	var numIdle = 0;
	var numBusy = 0;
	var numDeleted = 0;
	var numError = 0;

	// box-display
	var boxDisplay = "<div style='width: 600px; margin-top: 10px;'>";
	var coreList = Retina.keys(coreGroups).sort();
	for (var h=0; h<coreList.length; h++) {
	    boxDisplay += "<h5 style='clear: both;'>"+coreList[h]+" cores</h5>";
	    for (var i=0; i<coreGroups[coreList[h]].length; i++) {
		if (coreGroups[coreList[h]][i].Status == "active-idle") {
		    boxDisplay += widget.aweNode('info', i);
		    numIdle++;
		} else if (coreGroups[coreList[h]][i].Status == "active-busy") {
		    boxDisplay += widget.aweNode('success', i);
		    numBusy++;
		} else if (coreGroups[coreList[h]][i].Status == "suspend") {
		    boxDisplay += widget.aweNode('danger', i);
		    numError++;
		} else if (coreGroups[coreList[h]][i].Status == "deleted") {
		    boxDisplay += widget.aweNode('warning', i);
		    numDeleted++;
		}
	    }
	}
	boxDisplay += "</div><div style='clear: both;'></div>";

	// store history
	widget.history.push([ Date.now(), numIdle, numBusy, numError, numDeleted, parseInt(widget.aweServerData["jobs"].inprogress), parseInt(widget.aweServerData["jobs"].suspended), parseInt(widget.aweServerData["tasks"].queuing), parseInt(widget.aweServerData["tasks"].inprogress), parseInt(widget.aweServerData["tasks"].pending), parseInt(widget.aweServerData["tasks"].completed), parseInt(widget.aweServerData["tasks"].suspended), parseInt(widget.aweServerData["workunits"].queueing), parseInt(widget.aweServerData["workunits"].checkout), parseInt(widget.aweServerData["workunits"].suspended) ]);
	if (widget.history.length > widget.maxEntries) {
	    widget.history.shift();
	}
	
	// clients
	var html = "<div style='float: left;'><table style='font-size: 30px; font-weight: bold; text-align: center;'><tr><td style='color: black; width: 120px;'>"+data.length+"</td><td style='color: blue; width: 120px;'>"+numIdle+"</td><td style='color: green; width: 120px;'>"+numBusy+"</td><td style='color: orange; width: 120px;'>"+numDeleted+"</td><td style='color: red; width: 120px;'>"+numError+"</td></tr></table>";
	html += boxDisplay;
	html += "</div>";

	// jobs, tasks & workunits
	html += "<div style='float: left; height: 800px;'>\
<table class='table table-bordered'>\
<tr><th>jobs</th><th>"+widget.aweServerData["jobs"].all+"</th></tr>\
<tr><td>in-progress</td><td>"+widget.aweServerData["jobs"].inprogress+"</td></tr>\
<tr><td>suspended</td><td>"+widget.aweServerData["jobs"].suspended+"</td></tr>\
<tr><th>tasks</th><th>"+widget.aweServerData["tasks"].all+"</th></tr>\
<tr><td>queuing</td><td>"+widget.aweServerData["tasks"].queuing+"</td></tr>\
<tr><td>in-progress</td><td>"+widget.aweServerData["tasks"].inprogress+"</td></tr>\
<tr><td>pending</td><td>"+widget.aweServerData["tasks"].pending+"</td></tr>\
<tr><td>completed</td><td>"+widget.aweServerData["tasks"].completed+"</td></tr>\
<tr><td>suspended</td><td>"+widget.aweServerData["tasks"].suspended+"</td>\
<tr><th>workunits</th><th>"+widget.aweServerData["workunits"].all+"</th></tr>\
<tr><td>queuing</td><td>"+widget.aweServerData["workunits"].queueing+"</td></tr>\
<tr><td>checkout</td><td>"+widget.aweServerData["workunits"].checkout+"</td></tr>\
<tr><td>suspended</td><td>"+widget.aweServerData["workunits"].suspended+"</td></tr>\
</table>\
</div>";
	
	// graph
	html += "<div style='float: left;'><h5 style='text-align: center; margin-bottom: -17px; position: relative; right: 50px;'>client status</h5><div id='plotC'></div>";
	html += "<div style='float: left;'><h5 style='text-align: center; margin-bottom: -17px; position: relative; right: 50px;'>job status</h5><div id='plotJ'></div>";
	html += "<div style='float: left;'><h5 style='text-align: center; margin-bottom: -17px; position: relative; right: 50px;'>task status</h5><div id='plotT'></div>";
	html += "<div style='float: left;'><h5 style='text-align: center; margin-bottom: -17px; position: relative; right: 50px;'>workunit status</h5><div id='plotW'></div>";

	target.innerHTML = html;

	// draw the graphs
	var now = Date.now();
	var pointsClients = [ [], [], [], [] ];
	var pointsJobs = [ [], [] ];
	var pointsTasks = [ [], [], [], [], [] ];
	var pointsWorkunits = [ [], [], [] ];
	for (var i=0; i<widget.history.length; i++) {
	    for (var h=1; h<5; h++) {
		pointsClients[h - 1].push({ x: parseInt((now - widget.history[i][0]) / 1000), y: widget.history[i][h] });
	    }
	    for (var h=5; h<7; h++) {
		pointsJobs[h - 5].push({ x: parseInt((now - widget.history[i][0]) / 1000), y: widget.history[i][h] });
	    }
	    for (var h=7; h<12; h++) {
		pointsTasks[h - 7].push({ x: parseInt((now - widget.history[i][0]) / 1000), y: widget.history[i][h] });
	    }
	    for (var h=12; h<15; h++) {
		pointsWorkunits[h - 12].push({ x: parseInt((now - widget.history[i][0]) / 1000), y: widget.history[i][h] });
	    }
	}
	var pdataC = { series: [ { name: "idle", color: 'blue' },
				 { name: "busy", color: 'green' },
				 { name: "error", color: 'red' },
				 { name: "deleted", color: 'orange' } ],
		       points: pointsClients };
	var pdataJ = { series: [ { name: "in-progress", color: 'green' },
				 { name: "suspended", color: 'red' } ],
		       points: pointsJobs };
	var pdataT = { series: [ { name: "queuing", color: 'yellow' },
				 { name: "in-progress", color: 'blue' },
				 { name: "pending", color: 'orange' },
				 { name: "completed", color: 'green' },
				 { name: "suspended", color: 'red' } ],
		       points: pointsTasks };
	var pdataW = { series: [ { name: "queuing", color: 'blue' },
				 { name: "checkout", color: 'green' },
				 { name: "suspended", color: 'red' } ],
		       points: pointsWorkunits };

	if (! widget.hasOwnProperty('rendPlotC')) {
	    widget.rendPlotC = Retina.Renderer.create("plot", { target: document.getElementById('plotC'),
								show_legend: true,
								show_dots: false,
								x_scale: "int",
								y_scale: "int",
								height: 200,
								data: pdataC }).render();
	} else {
	    widget.rendPlotC.settings.data = pdataC;
	    widget.rendPlotC.settings.x_min = undefined;
	    widget.rendPlotC.settings.target = document.getElementById('plotC');
	    widget.rendPlotC.render();
	}
	if (! widget.hasOwnProperty('rendPlotJ')) {
	    widget.rendPlotJ = Retina.Renderer.create("plot", { target: document.getElementById('plotJ'),
								show_legend: true,
								x_scale: "int",
								y_scale: "int",
								height: 200,
								show_dots: false,
								data: pdataJ }).render();
	} else {
	    widget.rendPlotJ.settings.data = pdataJ;
	    widget.rendPlotJ.settings.x_min = undefined;
	    widget.rendPlotJ.settings.target = document.getElementById('plotJ');
	    widget.rendPlotJ.render();
	}
	if (! widget.hasOwnProperty('rendPlotT')) {
	    widget.rendPlotT = Retina.Renderer.create("plot", { target: document.getElementById('plotT'),
								show_legend: true,
								x_scale: "int",
								y_scale: "int",
								height: 200,
								show_dots: false,
								data: pdataT }).render();
	} else {
	    widget.rendPlotT.settings.data = pdataT;
	    widget.rendPlotT.settings.x_min = undefined;
	    widget.rendPlotT.settings.target = document.getElementById('plotT');
	    widget.rendPlotT.render();
	}
	if (! widget.hasOwnProperty('rendPlotW')) {
	    widget.rendPlotW = Retina.Renderer.create("plot", { target: document.getElementById('plotW'),
								show_legend: true,
								x_scale: "int",
								y_scale: "int",
								height: 200,
								show_dots: false,
								data: pdataW }).render();
	} else {
	    widget.rendPlotW.settings.data = pdataW;
	    widget.rendPlotW.settings.x_min = undefined;
	    widget.rendPlotW.settings.target = document.getElementById('plotW');
	    widget.rendPlotW.render();
	}
    };
    
    widget.getAWEDetails = function () {
	if (!(RetinaConfig.hasOwnProperty("alertPanelRefresh") && RetinaConfig.hasOwnProperty("alertPanelRefresh") == 0)) {
	    setTimeout(widget.getAWEDetails, RetinaConfig.alertPanelRefresh || 15000);
	}
	jQuery.ajax({ url: RetinaConfig.awe_ip+"/client",
		      headers: widget.authHeader,
		      dataType: "json",
		      success: function(data) {
			  jQuery.ajax({ url: RetinaConfig.awe_ip+"/queue",
					headers: widget.authHeader,
					dataType: "json",
					clients: data.data,
					success: function(data) {
					    var widget = Retina.WidgetInstances.awe_panel[1];
					    widget.aweClientData = this.clients;
					    var result = data.data;
					    var rows = result.split("\n");
					    widget.aweServerData = { "jobs": { "all": rows[1].match(/\d+/)[0],
									       "inprogress": rows[2].match(/\d+/)[0],
									       "suspended": rows[3].match(/\d+/)[0] },
								     "tasks": { "all": rows[4].match(/\d+/)[0],
										"queuing": rows[5].match(/\d+/)[0],
										"inprogress": rows[6].match(/\d+/)[0],
										"pending": rows[7].match(/\d+/)[0],
										"completed": rows[8].match(/\d+/)[0],
										"suspended": rows[9].match(/\d+/)[0] },
								     "workunits": { "all": rows[11].match(/\d+/)[0],
										    "queueing": rows[12].match(/\d+/)[0],
										    "checkout": rows[13].match(/\d+/)[0],
										    "suspended": rows[14].match(/\d+/)[0] } };
					    widget.showAWEDetails();
					}});
		      }
		    });
    };

    widget.aweNode = function (status, id) {
	return "<div class='alert alert-"+status+"' style='width: 24px; height: 24px; padding: 0px; margin-bottom: 5px; margin-right: 5px; float: left; cursor: pointer;' onclick='Retina.WidgetInstances.awe_panel[1].aweNodeDetail("+id+");'></div>";
    };

    widget.aweNodeDetail = function (nodeID) {
	
    };

    widget.resumeAllJobs = function () {
	var widget = Retina.WidgetInstances.awe_panel[1];
	jQuery.ajax({
	    method: "PUT",
	    dataType: "json",
	    headers: widget.authHeader, 
	    url: RetinaConfig["awe_ip"]+"/job?resumeall",
	    success: function (data) {
		Retina.WidgetInstances.awe_panel[1].test_components();
		alert('all jobs resumed');
	    }}).fail(function(xhr, error) {
		alert('failed to resume all jobs');
	    });
    };

    widget.resumeAllClients = function () {
	var widget = Retina.WidgetInstances.awe_panel[1];
	jQuery.ajax({
	    method: "PUT",
	    dataType: "json",
	    headers: widget.authHeader, 
	    url: RetinaConfig["awe_ip"]+"/client?resumeall",
	    success: function (data) {
		Retina.WidgetInstances.awe_panel[1].test_components();
		alert('all clients resumed');
	    }}).fail(function(xhr, error) {
		alert('failed to resume all clients');
	    });
    };

    widget.loginAction = function (action) {
	var widget = Retina.WidgetInstances.awe_panel[1];
	if (action.action == "login" && action.result == "success") {
	    widget.loggedIn = true;
	    widget.authHeader = { "Authorization": "OAuth "+action.token };
	} else {
	    widget.loggedIn = false;
	    widget.authHeader = {};
	}
	widget.display();
    };

})();
