(function () {
    var widget = Retina.Widget.extend({
        about: {
                title: "AWE Home",
                name: "awe_home",
                author: "Tobias Paczian",
                requires: []
        }
    });
    
    widget.setup = function () {
	    return [];
    };

    widget.display = function (wparams) {
	widget = Retina.WidgetInstances.awe_home[1];

	widget.target = wparams.target;
	widget.target.className = "mainview";
	widget.target.innerHTML = "\
<h2>Welcome to AWE!</h2>\
\
      <hr>\
\
      <p>AWE is a workflow engine that manages and executes scientific computing workflows or pipelines.</p>\
\
      <p>AWE is designed as a distributed system that contains a centralized server and multiple distributed clients. The server receives job submissions and parses jobs into tasks, splits tasks into workunits, and manages workunits in a queue. The AWE clients, running on distributed, heterogeneous computing resources, keep checking out workunits from the server queue and dispatching the workunits on the local computing resources.</p>\
\
      <p>AWE uses the Shock data management system to handle input and output data (retrieval, storage, splitting, and merge). AWE uses a RESTful API for communication between AWE components and with outside components such as Shock, the job submitter, and the status monitor.</p><div id='info'></div>";

	jQuery.ajax({ url: RetinaConfig["awe_ip"],
		      dataType: "json",
		      success: function(data) {
			  var html = "<h4 style='margin-top: 50px;'>Server Info</h4><table class='table' style='width: 600px;'>";
			  html += "<tr><td style='width: 200px;'><b>Server Version</b></td><td>"+data.version+"</td></tr>";
			  html += "<tr><td style='width: 200px;'><b>Administrative Contact</b></td><td>"+data.contact+"</td></tr>";
			  if (data.hasOwnProperty('title')) {
			      html += "<tr><td style='width: 200px;'><b>Title</b></td><td>"+data.title+"</td></tr>";
			      document.getElementById('serverTitle').innerHTML = data.title;
			  }
			  html += "</table>";
			  document.getElementById('info').innerHTML = html;
		      },
		      error: function(jqXHR, error) {
			  console.log(jqXHR);
			  console.log(error);
		      }
		    });

    };

})();
