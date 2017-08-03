(function () {
    var widget = Retina.Widget.extend({
        about: {
            title: "AWE TOP",
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
    
    widget.showAWEDetails = function (gp) {
	var widget = Retina.WidgetInstances.awe_top[1];
	
	var target = widget.target;
	target.style.margin = "50px";

	if (gp) {
	    widget.gp = gp;
	} else {
	    gp = widget.gp || "job";
	}

	var html = ["<h3>jobs</h3>"];
	html.push('<p>group by <select style="margin-bottom: 0px;" onchange="Retina.WidgetInstances.awe_top[1].showAWEDetails(this.options[this.selectedIndex].value);" id="grouping"><option'+(gp == 'job' ? ' selected=selected' : '')+'>job</option><option'+(gp == 'project' ? ' selected=selected' : '')+'>project</option><option'+(gp == 'user' ? ' selected=selected' : '')+'>user</option></select></p>');
	var cols = ['job', 'size', 'cores', 'machines', 'project', 'user', 'usage', 'submission', 'processing'];

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
	window.clients = clients;
	
	var jdata = {};
	var grouping = document.getElementById('grouping') ? document.getElementById('grouping').options[document.getElementById('grouping').selectedIndex].value : "job";
	for (var i=0; i<widget.aweWorkData.length; i++) {
	    var d = widget.aweWorkData[i];
	    if (d.info.pipeline.match(/^mgrast-prod/)) {
		var row = {};
		row.submission = d.info.submittime;
		row.processing = d.info.startedtime;
		row.project = d.info.project;
		row.user = d.info.user;
		row.job = d.info.name;
		row.size = parseInt(d.info.userattr.bp_count);

		if (! d.client) {
		    continue;
		}

		var g = d.info.name;
		if (grouping == 'project') {
		    g = d.info.project;
		} else if (grouping == 'user') {
		    g = d.info.user;
		}
		if (jdata.hasOwnProperty(g)) {
		    jdata[g].machines++;
		    jdata[g].cores += clients[d.client].cores;
		    jdata[g].usage = jdata[g].cores / totalCores * 100;
		} else {
		    row.cores = clients[d.client].cores;
		    row.usage = row.cores / totalCores * 100;
		    row.machines = 1;
		    jdata[g] = row;
		}
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
		    jtable[i][cols[h]] = jtable[i][cols[h]].baseSize();
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

    widget.loginAction = function (action) {
	var widget = Retina.WidgetInstances.awe_top[1];
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
