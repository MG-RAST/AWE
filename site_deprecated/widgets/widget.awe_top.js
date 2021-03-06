(function () {
    var widget = Retina.Widget.extend({
        about: {
            title: "AWE top",
            name: "awe_top",
            author: "Tobias Paczian",
            requires: [ "rgbcolor.js" ]
        }
    });

    widget.setup = function () {
	return [];
    };

    widget.sortattribute = "size";
    widget.sortdir = "asc";
    
    widget.display = function (wparams) {
	widget = Retina.WidgetInstances.awe_top[1];
	
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
	    widget.target.innerHTML = "<div style='text-align: center; margin-top: 300px;'><img src='Retina/images/waiting.gif'></div>";
	    widget.getAWEDetails();
	} else {
	    widget.target.innerHTML = "<p>authorization required</p>";
	}
    };
    
    widget.showAWEDetails = function (gp, type) {
	var widget = Retina.WidgetInstances.awe_top[1];
	
	var target = widget.target;
	target.style.margin = "50px";

	if (gp) {
	    widget.gp = gp;
	} else {
	    gp = widget.gp || "job";
	}

	if (type) {
	    widget.tabletype = type;
	} else {
	    type = widget.tabletype || "job";
	}

	var html = ["<div style='margin-bottom: 10px;'><div class='btn-group' data-toggle='buttons-radio'><button type='button' class='btn btn-large"+(type=='job'? " active" : '')+"' onclick='Retina.WidgetInstances.awe_top[1].sortattribute=\"job\";Retina.WidgetInstances.awe_top[1].showAWEDetails(null,\"job\");'>job</button><button type='button' class='btn btn-large"+(type=='task'? " active" : '')+"' onclick='Retina.WidgetInstances.awe_top[1].sortattribute=\"task\";Retina.WidgetInstances.awe_top[1].showAWEDetails(null,\"task\");'>task</button><button type='button' class='btn btn-large"+(type=='workunit'? " active" : '')+"' onclick='Retina.WidgetInstances.awe_top[1].sortattribute=\"workunit\";Retina.WidgetInstances.awe_top[1].showAWEDetails(null,\"workunit\");'>workunit</button></div></div>"];
	html.push('<p id="groupby">group by <select style="margin-bottom: 0px;" onchange="Retina.WidgetInstances.awe_top[1].showAWEDetails(this.options[this.selectedIndex].value);" id="grouping"><option'+(gp == 'job' ? ' selected=selected' : '')+'>job</option><option'+(gp == 'project' ? ' selected=selected' : '')+'>project</option><option'+(gp == 'user' ? ' selected=selected' : '')+'>user</option></select></p>');

	var cols = [];
	if (type == 'task') {
	    cols = ['task','pipeline','size','cores','machines','usage'];
	}  else if (type == 'workunit') {
	    cols = ['workunit','task','pipeline','size','cores','machines','usage'];
	} else if (type == 'job') {
	    cols = ['job', 'project', 'user', 'pipeline', 'size', 'cores', 'machines', 'usage', 'submission', 'processing'];
	}

	html.push('<table class="table" style="margin-left: 10%; width: 80%;"><tr>');
	for (var i=0; i<cols.length; i++) {
	    html.push('<th style="cursor: pointer;" onclick="Retina.WidgetInstances.awe_top[1].setSort(\''+cols[i]+'\');">'+cols[i]+'</th>');
	};

	var clients = {};
	var totalCores = 0;
	for (var i=0; i<widget.aweClientData.length; i++) {
	    clients[widget.aweClientData[i].id] = widget.aweClientData[i];
	    totalCores += widget.aweClientData[i].cores;
	}
	
	var jdata = {};
	var grouping = document.getElementById('grouping') ? document.getElementById('grouping').options[document.getElementById('grouping').selectedIndex].value : "job";
	for (var i=0; i<widget.aweWorkData.length; i++) {
	    var d = widget.aweWorkData[i];
	    var row = {};
	    row.workunit = d.wuid;
	    row.task = d.cmd.name;
	    row.pipeline = d.info.pipeline;
	    row.submission = d.info.submittime;
	    row.processing = d.info.startedtime;
	    row.project = d.info.project;
	    row.user = d.info.user;
	    row.job = d.info.name;
	    row.size = 0;
	    for (var h=0; h<d.inputs.length; h++) {
		row.size += d.inputs[h].size;
	    }
	    
	    if (! d.client) {
		continue;
	    }
	    
	    var g = d.info.name;
	    if (grouping == 'project') {
		g = d.info.project;
	    } else if (grouping == 'user') {
		g = d.info.user;
	    }
	    if (type == 'task') {
		g = row.task;
	    }
	    if (jdata.hasOwnProperty(g)) {
		jdata[g].machines++;
		jdata[g].cores += clients[d.client].cores;
		jdata[g].usage = jdata[g].cores / totalCores * 100;
		jdata[g].size += row.size;
	    } else {
		row.cores = clients[d.client].cores;
		row.usage = row.cores / totalCores * 100;
		row.machines = 1;
		jdata[g] = row;
	    }
	}
	var jtable = [];
	var jobs = Retina.keys(jdata);
	for (var i=0; i<jobs.length; i++) {
	    jtable.push(jdata[jobs[i]]);
	}
	
	var sortattr = widget.sortattribute;
	jtable.sort(Retina.propSort(sortattr, widget.sortdir == 'asc' ? true : false));

	for (var i=0; i<jtable.length; i++) {
	    var row = ['<tr>'];
	    for (var h=0; h<cols.length; h++) {
		if (cols[h] == 'size') {
		    jtable[i][cols[h]] = jtable[i][cols[h]].byteSize();
		}
		if (cols[h] == 'usage') {
		    jtable[i][cols[h]] = jtable[i][cols[h]].formatString(3) + "%";
		}
		row.push('<td>'+jtable[i][cols[h]]+'</td>');
	    }
	    row.push('</tr>');
	    html.push(row.join(''));
	}
	
	html.push('</table>');
		
	target.innerHTML = html.join("");

	if (type == 'task') {
	    document.getElementById('groupby').style.display = 'none';
	}  else if (type == 'workunit') {
	    document.getElementById('groupby').style.display = 'none';
	} else if (type == 'job') {
	    document.getElementById('groupby').style.display = '';
	}
    };

    widget.setSort = function (field) {
	var widget = this;

	if (field == widget.sortattribute) {
	    if (widget.sortdir == 'asc') {
		widget.sortdir = 'desc';
	    } else {
		widget.sortdir = 'asc';
	    }
	} else {
	    widget.sortattribute = field;
	}

	widget.showAWEDetails();
    };
    
    widget.getAWEDetails = function () {
	if (!(RetinaConfig.hasOwnProperty("alertPanelRefresh") && RetinaConfig.hasOwnProperty("alertPanelRefresh") == 0)) {
	    setTimeout(widget.getAWEDetails, RetinaConfig.alertPanelRefresh || 60000);
	}
	jQuery.ajax({ url: RetinaConfig.awe_ip+"/client",
		      headers: widget.authHeader,
		      dataType: "json",
		      success: function(data) {
			  var widget = Retina.WidgetInstances.awe_top[1];
			  widget.aweClientData = data.data;
			  jQuery.ajax({ url: RetinaConfig.awe_ip+"/work?query&state=checkout&limit=1000",
					headers: widget.authHeader,
					dataType: "json",
					success: function(data) {
					    var widget = Retina.WidgetInstances.awe_top[1];
					    widget.aweWorkData = data.data;
					    widget.showAWEDetails();
					}});
		      }
		    });
    };

})();
