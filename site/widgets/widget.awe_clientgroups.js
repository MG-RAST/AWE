(function () {
    var widget = Retina.Widget.extend({
        about: {
                title: "AWE Clientgroups",
                name: "awe_clientgroups",
                author: "Tobias Paczian",
                requires: []
        }
    });
    
    widget.setup = function () {
	    return [];
    };

    widget.authHeader = {};

    widget.display = function (wparams) {
	widget = Retina.WidgetInstances.awe_clientgroups[1];

	widget.target = widget.target || wparams.target;
	widget.target.className = "mainview";
	widget.target.style.width = "800px";
	widget.target.innerHTML = "";

	if (widget.authHeader.hasOwnProperty("Authorization")) {
	    widget.target.innerHTML += "<div id='cgroup-content'></div>";
	    widget.showClientgroups();
	} else {
	    widget.target.innerHTML += "<h3>Authentication required</h3><p>You need to be logged in to use the clientgroup management.</p>";
	}
    };

    // get the currently available clientgroups from the api and show them upon success
    widget.showClientgroups = function () {
	var widget = Retina.WidgetInstances.awe_clientgroups[1];

	jQuery.ajax( { dataType: "json",
		       url: RetinaConfig["awe_ip"]+"/cgroup?limit=1000",
		       headers: widget.authHeader,
		       error: function (xhr, error) {
			   var widget = Retina.WidgetInstances.awe_clientgroups[1];
			   widget.renderClientgroups([]);
		       },
		       success: function (data) {
			   var widget = Retina.WidgetInstances.awe_clientgroups[1];
			   
			   // store current clientgroup data in stm
			   stm.DataStore.clientgroups = {};
			   for (var i=0; i<data.data.length; i++) {
			       stm.DataStore.clientgroups[data.data[i].id] = data.data[i];
			   }
			   
			   // render the clientgroup data
			   widget.renderClientgroups(data.data);
		       }
	    });
    };

    // request a new clientgroup
    widget.requestClientgroup = function (name) {
	var widget = Retina.WidgetInstances.awe_clientgroups[1];

	jQuery.ajax( { dataType: "json",
		       method: "POST",
		       url: RetinaConfig["awe_ip"]+"/cgroup/"+name,
		       headers: widget.authHeader,
		       error: function (xhr, error) {
			   var widget = Retina.WidgetInstances.awe_clientgroups[1];
			   alert('clientgroup creation failed');
		       },
		       success: function (data) {
			   var widget = Retina.WidgetInstances.awe_clientgroups[1];
			   widget.showClientgroups();
		       }
	    });
    };

    // render the retrieved clientgroups
    widget.renderClientgroups = function (data) {
	var widget = Retina.WidgetInstances.awe_clientgroups[1];
	var target = document.getElementById('cgroup-content');
	
	var html = "";

	// request new group
	html += "<h4>request new clientgroup</h3>";
	html += '\
<div class="input-append">\
  <input type="text" value=""><button class="btn" onclick="Retina.WidgetInstances.awe_clientgroups[1].requestClientgroup(this.previousSibling.value);">request</button>\
</div>\
';
	
	// get existing clientgroups
	html += "<h4>existing clientgroups</h3>";

	if (data.length) {
	    
	    html += "<table class='table'>";
	    html += "<tr><th>name</th><th>IP</th><th>created</th><th>modified</th><th>expires</th><th>details</th><th>status</th></tr>";
	    for (var i=0; i<data.length; i++) {
		var status = "<span style='color: green;'>active</span>";
		if (Date.parse(data[i].expiration) <= new Date().getTime()) {
		    status = "<span style='color: red;'>expired</span>";
		}
		html += "<tr><td>"+data[i].name+"</td><td>"+data[i].ip_cidr+"</td><td>"+data[i].created_on+"</td><td>"+data[i].last_modified+"</td><td>"+data[i].expiration+"</td><td style='text-align: center;'><button class='btn btn-mini' onclick='Retina.WidgetInstances.awe_clientgroups[1].clientgroupDetails(\""+data[i].id+"\");'><i class='icon-cog'></i></button></td><td>"+status+"</td></tr>";
	    }
	    html += "</table>";
	} else {
	    html += "<p>no clientgroups found</p>";
	}

	// details
	html += "<div id='clientgroupDetails'></div>";

	target.innerHTML = html;
    };

    // show the detail data of a clientgroup
    widget.clientgroupDetails = function (id) {
	var widget = Retina.WidgetInstances.awe_clientgroups[1];
	var target = document.getElementById('clientgroupDetails');
	
	var cgroup = stm.DataStore.clientgroups[id];

	var status = (Date.parse(cgroup.expiration) <= new Date().getTime()) ? "<span style='color: red;'>expired</span>" : "<span style='color: green;'>active</span>";

	var html = "<h4>clientgroup: "+cgroup.name+"</h4>";
	html += '\
<button class="btn btn-small" style="margin-right: 15px;" onclick="Retina.WidgetInstances.awe_clientgroups[1].clientgroupUpdateToken(\''+id+'\', \'delete\');">delete token</button><button class="btn btn-small" onclick="Retina.WidgetInstances.awe_clientgroups[1].clientgroupUpdateToken(\''+id+'\', \'new\');">new token</button>\
<table class="table table-hover" style="table-layout: fixed; word-wrap: break-word; margin-top: 15px;">\
  <tr><td class="span2"><b>status</b></td><td>'+status+'</td></tr>\
  <tr><td><b>created</b></td><td>'+cgroup.created_on+'</td></tr>\
  <tr><td><b>modified</b></td><td>'+cgroup.last_modified+'</td></tr>\
  <tr><td><b>expires</b></td><td>'+cgroup.expiration+'</td></tr>\
  <tr><td><b>ID</b></td><td>'+cgroup.id+'</td></tr>\
  <tr><td><b>token</b></td><td>'+cgroup.token+'</td></tr>\
</table>\
<h5>permissions</h5>\
<div id="clientgroupPermissions"><img src="Retina/images/loading.gif"></div>\
';
	
	jQuery.ajax( { dataType: "json",
		       url: RetinaConfig["awe_ip"]+"/cgroup/"+cgroup.id+"/acl",
		       headers: widget.authHeader,
		       error: function (xhr, error) {
			   var widget = Retina.WidgetInstances.awe_clientgroups[1];
			   document.getElementById('clientgroupPermissions').innerHTML = "- permission data not available -";
		       },
		       success: function (data) {
			   var widget = Retina.WidgetInstances.awe_clientgroups[1];
			   document.getElementById('clientgroupPermissions').innerHTML = "<pre>"+JSON.stringify(data.data, null, 1)+"</pre>";
		       }
		     });

	target.innerHTML = html;
    };

    widget.clientgroupUpdateToken = function (id, func) {
	var widget = Retina.WidgetInstances.awe_clientgroups[1];
	var method = "PUT";
	if (func == "delete") {
	    method = "DELETE";
	}

	jQuery.ajax( { 
	    method: method,
	    dataType: "json",
	    url: RetinaConfig["awe_ip"]+"/cgroup/"+id+"/token",
	    headers: widget.authHeader,
	    error: function (xhr, error) {
		var widget = Retina.WidgetInstances.awe_clientgroups[1];
		alert('the requested function on the token could not be performed');
		console.log(xhr);
		console.log(error);
	    },
	    success: function (data) {
		var widget = Retina.WidgetInstances.awe_clientgroups[1];
		
		widget.showClientgroups();
	    }
	});
    };

    // update the views if the auth status changes
    widget.loginAction = function (action) {
	var widget = Retina.WidgetInstances.awe_clientgroups[1];
	if (action.action == "login" && action.result == "success") {
	    widget.authHeader = { "Authorization": "OAuth "+action.token };
	} else {
	    widget.authHeader = {};
	}
	widget.display();
    };

})();
