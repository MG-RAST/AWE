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
			   widget.renderClientgroups(data.data);
		       }
	    });
    };

    widget.requestClientgroup = function (name) {
	var widget = Retina.WidgetInstances.awe_clientgroups[1];

	jQuery.ajax( { dataType: "json",
		       url: RetinaConfig["awe_ip"]+"/cgroup/"+name,
		       headers: widget.authHeader,
		       error: function (xhr, error) {
			   var widget = Retina.WidgetInstances.awe_clientgroups[1];
			   console.log(xhr);
			   console.log(error);
		       },
		       success: function (data) {
			   var widget = Retina.WidgetInstances.awe_clientgroups[1];
			   console.log(data);
		       }
	    });
    };


    widget.renderClientgroups = function (data) {
	var widget = Retina.WidgetInstances.awe_clientgroups[1];
	var target = document.getElementById('cgroup-content');
	
	var html = "";
	
	// get existing clientgroups
	html += "<h3>existing clientgroups</h3>";

	if (data.length) {
	    
	    html += "<table class='table'>";
	    html += "<tr><th>name</th><th>IP</th><th>created</th><th>expires</th><th>modified</th><th>ID</th><th>token</th></tr>";
	    for (var i=0; i<data.length; i++) {
		html += "<tr><td>"+data[i].name+"</td><td>"+data[i].ip_cidr+"</td><td>"+data[i].created_on+"</td><td>"+data[i].expiration+"</td><td>"+data[i].last_modified+"</td><td><button class='btn btn-mini' onclick='alert(\""+data[i].id+"\");'><i class='icon-cog'></i></button></td><td><button class='btn btn-mini' onclick='alert(\""+data[i].token+"\");'><i class='icon-cog'></i></button></td></tr>";
	    }
	    html += "</table>";
	} else {
	    html += "<p>no clientgroups found</p>";
	}

	html += "<h3>request new clientgroup</h3>";

	html += '\
<input type="text" value="hans"><button class="btn" onclick="Retina.WidgetInstances.awe_clientgroups[1].requestClientgroup(this.previousSibling.value);">create</button>\
';

	target.innerHTML = html;
    };

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
