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

	jQuery.ajax({ url: RetinaConfig["awe_ip"],
		      dataType: "json",
		      success: function(data) {
			  if (data.hasOwnProperty('title')) {
			      document.getElementById('serverTitle').innerHTML = data.title;
			  }
		      }
		    });
	
	if (! RetinaConfig.authentication || widget.loggedIn) {
	    widget.target.innerHTML = "<div style='position: absolute; left: 540px; top: 90px;' id='control'><select style='margin-bottom: 0px; width: 100px; padding: 0px; height: 25px;' onchange='Retina.WidgetInstances.awe_panel[1].grouping=this.options[this.selectedIndex].value;Retina.WidgetInstances.awe_panel[1].showAWEDetails();'><option>cores</option><option>group</option><option>domain</option><option>instance_type</option></select></div><div id='awe_details' style='margin-left: 50px; margin-top: 50px;'></div><div style='float: left; padding: 20px;' id='detail'></div>";
	    widget.getAWEDetails();
	} else {
	    widget.target.innerHTML = "<p>authorization required</p>";
	}
    };
    
    widget.showAWEDetails = function () {
	var widget = Retina.WidgetInstances.awe_panel[1];
	
	var target = document.getElementById('awe_details');

	var grouping = widget.grouping || "cores";
	var groups = {};
	var groupOrder = [];
	
	var data = widget.aweClientData;
	var clientIndex = {};
	data = data.sort(Retina.propSort('id'));
	for (var i=0; i<data.length; i++) {
	    data[i].index = i;
	    clientIndex[data[i].id] = i;
	    if (! groups.hasOwnProperty(data[i][grouping])) {
		groups[data[i][grouping]] = [];
		groupOrder.push(data[i][grouping]);
	    }
	    groups[data[i][grouping]].push(data[i]);
	}
	if (groupOrder.length) {
	    if (typeof groupOrder[0].match == 'function') {
		groupOrder = groupOrder.sort();
	    } else {
		groupOrder = groupOrder.sort(Retina.Numsort);
	    }
	}

	widget.clientIndex = clientIndex;

	var apps = {};
	var pipelines = {};
	var wdata = widget.aweWorkData;
	for (var i=0; i<wdata.length; i++) {
	    if (! apps.hasOwnProperty(wdata[i].cmd.name)) {
		apps[wdata[i].cmd.name] = [];
	    }
	    if (! pipelines.hasOwnProperty(wdata[i].info.pipeline)) {
		pipelines[wdata[i].info.pipeline] = [];
	    }
	    wdata[i].client = wdata[i].client.replace(/^<[^>]+>([^<]+)<\/.+$/, '$1');
	    apps[wdata[i].cmd.name].push(wdata[i].client);
	    pipelines[wdata[i].info.pipeline].push(wdata[i].client);
	    if (clientIndex[wdata[i].client] && data[clientIndex[wdata[i].client]]) {
		data[clientIndex[wdata[i].client]].current_work[wdata[i].wuid] = "<span class='href' onclick='Retina.WidgetInstances.awe_panel[1].aweWorkunitDetail("+i+");'>"+wdata[i].cmd.name+"</span>";
		
		if (widget.currentApp && (widget.currentApp == wdata[i].cmd.name || widget.currentApp == wdata[i].info.pipeline) ) {
		    data[clientIndex[wdata[i].client]].highlight = true;
		} else {
		    data[clientIndex[wdata[i].client]].highlight = false;
		}
		wdata[i].client = "<span class='href' onclick='Retina.WidgetInstances.awe_panel[1].aweNodeDetail("+clientIndex[wdata[i].client]+");'>"+wdata[i].client+"</span>";
	    }
	}
	
	// client counts
	var numIdle = 0;
	var numBusy = 0;
	var numDeleted = 0;
	var numError = 0;

	// box-display
	var boxDisplay = "<div style='width: 600px; margin-top: 10px;'>";
	for (var h=0; h<groupOrder.length; h++) {
	    boxDisplay += "<h5 style='clear: both;'>"+grouping+": "+groupOrder[h]+"</h5>";
	    for (var i=0; i<groups[groupOrder[h]].length; i++) {
		if (groups[groupOrder[h]][i].Status == "active-idle") {
		    boxDisplay += widget.aweNode('info', groups[groupOrder[h]][i]);
		    numIdle++;
		} else if (groups[groupOrder[h]][i].Status == "active-busy") {
		    boxDisplay += widget.aweNode('success', groups[groupOrder[h]][i]);
		    numBusy++;
		} else if (groups[groupOrder[h]][i].Status == "suspend") {
		    boxDisplay += widget.aweNode('danger', groups[groupOrder[h]][i]);
		    numError++;
		} else if (groups[groupOrder[h]][i].Status == "deleted") {
		    boxDisplay += widget.aweNode('warning', groups[groupOrder[h]][i]);
		    numDeleted++;
		}
	    }
	}
	boxDisplay += "</div><div style='clear: both;'></div>";
	
	// clients
	var html = ["<div style='float: left;'><table style='font-size: 30px; font-weight: bold; text-align: center;'><tr><td style='color: black; width: 120px;' title='total number of clients'>"+data.length+"</td><td style='color: blue; width: 120px;' title='number of idle clients'>"+numIdle+"</td><td style='color: green; width: 120px;' title='number of busy clients'>"+numBusy+"</td><td style='color: orange; width: 120px;' title='number of deleted clients'>"+numDeleted+"</td><td style='color: red; width: 120px;' title='number of error clients'>"+numError+"</td></tr></table>"];
	html.push( boxDisplay );
	html.push("</div>");

	html.push("<div style='float: left; height: 800px;'><table class='table table-condensed table-bordered'><tr><th colspan='2' onclick='Retina.WidgetInstances.awe_panel[1].currentApp=null;Retina.WidgetInstances.awe_panel[1].showAWEDetails();' style='cursor: pointer;'>Running Applications</th></tr>");
	var appNames = Retina.keys(apps).sort();
	for (var i=0; i<appNames.length; i++) {
	    html.push("<tr"+(widget.currentApp && widget.currentApp==appNames[i] ? " class='info'" : "")+"><td onclick='Retina.WidgetInstances.awe_panel[1].currentApp=\""+appNames[i]+"\";Retina.WidgetInstances.awe_panel[1].showAWEDetails();' style='cursor: pointer;'>"+appNames[i]+"</td><td style='padding-left: 10px;'>"+apps[appNames[i]].length+"</td></tr>");
	}
	html.push("<tr><th colspan='2' onclick='Retina.WidgetInstances.awe_panel[1].currentApp=null;Retina.WidgetInstances.awe_panel[1].showAWEDetails();' style='cursor: pointer;'>Running Pipelines</th></tr>");
	var pipeNames = Retina.keys(pipelines).sort();
	for (var i=0; i<pipeNames.length; i++) {
	    html.push("<tr"+(widget.currentApp && widget.currentApp==pipeNames[i] ? " class='info'" : "")+"><td onclick='Retina.WidgetInstances.awe_panel[1].currentApp=\""+pipeNames[i]+"\";Retina.WidgetInstances.awe_panel[1].showAWEDetails();' style='cursor: pointer;'>"+pipeNames[i]+"</td><td style='padding-left: 10px;'>"+pipelines[pipeNames[i]].length+"</td></tr>");
	}
	html.push("</table></div>");
		
	target.innerHTML = html.join("");

    };
    
    widget.getAWEDetails = function () {
	if (!(RetinaConfig.hasOwnProperty("alertPanelRefresh") && RetinaConfig.hasOwnProperty("alertPanelRefresh") == 0)) {
	    setTimeout(widget.getAWEDetails, RetinaConfig.alertPanelRefresh || 60000);
	}
	jQuery.ajax({ url: RetinaConfig.awe_ip+"/client",
		      headers: widget.authHeader,
		      dataType: "json",
		      success: function(data) {
			  var widget = Retina.WidgetInstances.awe_panel[1];
			  widget.aweClientData = data.data;
			  jQuery.ajax({ url: RetinaConfig.awe_ip+"/work?query&state=checkout&limit=1000",
					headers: widget.authHeader,
					dataType: "json",
					success: function(data) {
					    var widget = Retina.WidgetInstances.awe_panel[1];
					    widget.aweWorkData = data.data;
					    widget.showAWEDetails();
					}});
		      }
		    });
    };

    widget.aweNode = function (status, data) {
	return "<div class='box alert alert-"+status+"' "+(data.highlight ? " style='box-shadow: 0 0 1em blue;'" : "")+"onclick='Retina.WidgetInstances.awe_panel[1].aweNodeDetail("+data.index+");'>"+Retina.keys(data.current_work).length+"</div>";
    };

    widget.aweNodeDetail = function (nodeIndex) {
	var widget = this;
	
	var node = widget.aweClientData[nodeIndex];

	document.getElementById('detail').innerHTML = '<div class="json">'+JSON.stringify(node, true, 2)+'</div>';
    };

    widget.aweWorkunitDetail = function (id) {
	var widget = this;

	var node = widget.aweWorkData[id];

	document.getElementById('detail').innerHTML = '<div class="json" style="width: 600px;">'+JSON.stringify(node, true, 2)+'</div>';	
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
