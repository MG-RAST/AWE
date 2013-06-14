(function () {
    var root = this;
    var stm = root.stm = {};

    // global variables
    stm.DataStore = [];
    stm.DataRepositories = [];
    stm.TypeData = [];
    stm.DataRepositoryDefault = null;
    stm.DataRepositoriesCount = 0;
    stm.SourceOrigin = "*";
    stm.TargetOrigin = "*";
    stm.Authentication = null;
    stm.Config = RetinaConfig;

    // receive messages sent from other frames
    //window.addEventListener("message", receiveMessage, false);
    var receiveMessage = stm.receiveMessage = function (event) {

	// do not caputre the event if the allowed origin does not match
	if (stm.SourceOrigin != '*' && event.origin !== stm.SourceOrigin) { return; }
	
	// parse the data into an object
	var data = event.data;
	if (typeof(data) == 'object' && data.hasOwnProperty('type')) {
	    var type = data.type;
	    // check what kind of message this is
	    if (type == 'data') {
		// try to load the data
		stm.load_data(data.data);
	    } else if (type == 'action') {
		data.data = data.data.replace(/##/g, "\\'").replace(/!!/g, '"');
		eval(data.data);
	    } else if (type == 'html') {
		document.getElementById(data.data.target).innerHTML = data.data.data;
	    }
	}
	// alert if the data is not valid json
	else {
	    alert('invalid message received');
	}
    };

    // send data to another iframe
    stm.send_message = function (frame, data, type, no_clear) {
	// if frame is a string, interpret it as an id
	if (typeof(frame) == 'string') {
	    frame = document.getElementById(frame);
	}

	if (! (frame.parent === frame)) {
	    // check if frame is an iframe
	    if (typeof(frame.contentWindow) !== 'undefined') {
		frame = frame.contentWindow;	    
	    }
	}

	// check if frame is now a window object
	if (typeof(frame.postMessage) == 'undefined') {
	    alert('invalid target object');
	}

	// check if a type was passed
	if (! type) {
	    type = 'data';
	}

	if (type == 'data') {

	    // check if data is an array
	    if (typeof(data.length) == 'undefined') {
		data = [ data ];
	    }
	    if (typeof(data[0].type) == 'undefined') {
		alert('invalid data');
	    } else {
		type = data[0].type;
	    }

	}

	// send out the data
	frame.postMessage({ 'type': type, 'data': data }, stm.TargetOrigin);
    };

    // get / set a repository, or get all repos if no argument is passed
    stm.repository = function (name, value) {
	if (typeof value !== 'undefined' &&
	    typeof name  !== 'undefined' ) {
	    return stm.DataRepositories[name] = value;
	} else if (typeof name !== 'undefined') {
	    return stm.DataRepositories[name];
	} else {
	    return stm.DataRepositories;
	}
    };
    stm.repositories = function () { return stm.repository(); };
    
    // set up / reset the DataHandler, adding initial repositories
    stm.init = function (repo, nocheck, name) {
	    stm.DataStore = [];
	    stm.TypeData = [];
	    stm.TypeData['object_count'] = [];
	    stm.TypeData['type_count'] = 0;
	    CallbackList = [];
	    stm.DataRepositories = [];
	    stm.DataRepositoriesCount = 0;
	    stm.DataRepositoryDefault = null;
	    if (repo) {
	        return stm.add_repository(repo, null, nocheck, name);
	    }
    };
    
    // generic data loader
    // given a DOM id, interprets the innerHTML of the element as JSON data and loads it into the DataStore
    // given a JSON data structure, loads it into the DataStore
    stm.load_data = function (params) {
	var id_or_data = params['data'] || [];
	var no_clear = params['no_clear'] || 0;
	var type = params['type'];
	var new_data;
	if (typeof(id_or_data) == 'string') {
	    var elem = document.getElementById(id);
	    if (elem) {
		new_data = JSON.parse(elem.innerHTML);
		if (!no_clear) {
		    document.getElementById(id).innerHTML = "";
		}
	    }
	} else {
	    new_data = id_or_data;
	}
	
	if (new_data) {
	    if (! new_data.length) {
		if (new_data.hasOwnProperty("next") && new_data.hasOwnProperty("data")) {
		    new_data = new_data.data;
		} else {
		    new_data = [ new_data ];
		}
	    }
	    if (typeof(new_data[0]) != 'object') {
		var dataids = [];
		for (i=0; i<new_data.length; i++) {
		    dataids.push( { 'id': new_data[i] } );
		}
		new_data = dataids;
	    }
	    if (! stm.TypeData['object_count'][type]) {
		stm.DataStore[type] = [];
		stm.TypeData['type_count']++;
		stm.TypeData['object_count'][type] = 0;
	    }
	    for (var i = 0; i < new_data.length; i++) {	
		if (!stm.DataStore[type][new_data[i].id]) {
		    stm.TypeData['object_count'][type]++;
		}
		stm.DataStore[type][new_data[i].id] = new_data[i];
	    }
	}
    };
    
    // adds / replaces a repository in the stm.DataRepositories list
    stm.add_repository = function (repository, resolve_resources, offline_mode, repository_name) {
	    if (repository) {
	        if (offline_mode) {
	            repository_name = repository_name || 'default';
		        stm.DataRepositories[repository_name] = { url: repository, name: repository_name };
		        stm.DataRepositoriesCount++;
		        if (stm.DataRepositoryDefault == null) {
		            stm.DataRepositoryDefault = stm.DataRepositories[repository_name];
		        }
	        } else {
		        var promises = [];
		        var promise = jQuery.Deferred();
		        var repository_url = repository;
		        promises.push(jQuery.getJSON(repository, function (data) {
		            repository = data;
		            repository_name = repository_name || repository.service;
		            if (! repository.url) {
		                repository.url = repository_url;
		            }
		            stm.DataRepositories[repository_name] = repository;
		            if (resolve_resources) {
			            for (var i=0; i<repository.resources.length; i++) {
			                promises.push(jQuery.getJSON(repository.resources[i].url, function (data) {
				                stm.TypeData[data.name] = data;
			                }));
			            }
		            }
		            stm.DataRepositoriesCount++;
		            if (stm.DataRepositoryDefault == null) {
			            stm.DataRepositoryDefault = stm.DataRepositories[repository_name];
		            }
		        }));
		        jQuery.when.apply(this, promises).then(function() {
		            promise.resolve();
		        });
		        return promise;
	        }
	    }
    };
    
    // removes a repository from the stm.DataRepositories list
    stm.remove_repository = function (id) {
	if (id && stm.DataRepositories[id]) {
	    stm.DataRepositories[id] = null;
	    stm.DataRepositoriesCount--;
	    if (stm.DataRepositoriesCount == 1) {
		for (var i in stm.DataRepositories) {
		    stm.DataRepositoryDefault = stm.DataRepositories[i];
		}
	    }
	}
    };
    
    // sets the default repository
    stm.default_repository = function (id) {
	if (id && stm.DataRepositories[id]) {
	    stm.DataRepositoryDefault = stm.DataRepositories[id];
	}
	return stm.DataRepositoryDefault;
    };
    
    // event handler for an input type file element, which interprets the selected file(s)
    // as JSON data and loads them into the DataStore
    stm.file_upload = function (evt, callback_function, callback_parameters) {
	var files = evt.target.files;
	
	if (files.length) {
	    for (var i = 0; i < files.length; i++) {
		var f = files[i];
		var reader = new FileReader();
		reader.onload = (function(theFile) {
		    return function(e) {
			var new_data = JSON.parse(e.target.result.toString().replace(/\n/g, ""));
			for (var h in new_data) {
			    if (new_data.hasOwnProperty(h)) {
				stm.load_data({ data: new_data[h], type: h});
			    }
			}
			if (typeof(callback_function) == 'function') {
			    callback_function.call(null, callback_parameters);
			}
		    };
		})(f);
		reader.readAsText(f);
	    }
	}
    };
    
    // client side data requestor
    // initiates data retrieval from a resource, saving callback functions /
    // paramters
    stm.get_objects = function (params) {
	var promise = jQuery.Deferred();

	var repo = params['repository'];
	if (repo) {
	    repo = stm.repository(repo);
	} else {
	    repo = stm.default_repository();
	}

	var type = params['type'];
	var id = params['id'];
	if (params.hasOwnProperty('return_type') && (params.return_type == 'search')) {
	    id = '"'+id+'"';
	}
	if (id) {
	    id = '/'+id;
	} else {
	    id = '';
	}
	var options = params['options'];
	
	var query_params = "";
	if (options) {
	    query_params = "?";
	    for (var i in options) {
		    query_params += i + '=' + options[i] + '&';
	    }
	    query_params = query_params.slice(0,-1);
	}
	
	var base_url = repo.url;
	base_url += "/" + type + id + query_params;
	var xhr = new XMLHttpRequest(); 
	xhr.addEventListener("progress", updateProgress, false);
	if ("withCredentials" in xhr) {
	    xhr.open('GET', base_url, true);
	} else if (typeof XDomainRequest != "undefined") {
	    xhr = new XDomainRequest();
	    xhr.open('GET', base_url);
	} else {
	    alert("your browser does not support CORS requests");
	    console.log("your browser does not support CORS requests");
	    return;
	}
	xhr.onload = function() {
	    var progressIndicator = document.getElementById('progressIndicator');
	    if (progressIndicator) {
		document.getElementById('progressBar').innerHTML = "waiting for respose...";
		//progressIndicator.style.display = "none";
	    }
	    if (params.hasOwnProperty('return_type')) {
		switch (params.return_type) {
		case 'text':
		    var d = {};
		    d['id'] = params['id'];
		    d['data'] = xhr.responseText;
		    stm.load_data({ "data": d, "type": type });
		    break;
		case 'shock':
		    var d = JSON.parse(xhr.responseText);
		    if (d.error == null) {
			stm.load_data({ "data": d.data, "type": type });
		    } else {
			alert(d.error);
			console.log(d);
		    }
		    break;
		case 'search':
		    var d = JSON.parse(xhr.responseText);
		    if (d.found && d.found > 0 && d.body && d.body.length) {
			for (i=0;i<d.body.length;i++) {
			    if (d.body[i].hasOwnProperty('gid')) { d.body[i].id = d.body[i].gid; }
			    if (d.body[i].hasOwnProperty('fid')) { d.body[i].id = d.body[i].fid; }
			    if (d.body[i].hasOwnProperty('kbfid')) { d.body[i].id = d.body[i].kbfid; }
			}
			stm.load_data({ "data": d.body, "type": type });
		    } else {
			alert('could not retrieve requested data');
			console.log(d);
		    }
		    break;
		}
	    } else {
		stm.load_data({ "data": JSON.parse(xhr.responseText), "type": type });
	    }

	    promise.resolve();
	};
	
	xhr.onerror = function() {
	    alert('The data retrieval failed.');
	    console.log("data retrieval failed");
	    return;
	};
	
	xhr.onabort = function() {
	    console.log("data retrieval was aborted");
	    return;
	};

	if (stm.Authentication) {
	    xhr.setRequestHeader('AUTH', stm.Authentication);
	}
	
	var progressIndicator = document.getElementById('progressIndicator');
	if (progressIndicator) {
	    progressIndicator.style.display = "";
	    document.getElementById('progressBar').innerHTML = "requesting data...";
	}
	
	xhr.send();

	return promise;
    };
    
    function updateProgress (e) {
	var progressBar = document.getElementById('progressBar');
	if (progressBar) {
	    document.getElementById('progressIndicator').style.display = "";	    
	    if (e.loaded) {
		progressBar.innerHTML = "received... "+pretty_size(e.loaded);
	    }
	}
    }
    
    function pretty_size (size) {
	var magnitude = "B";
	if (size > 999) {
	    size = size / 1024;
	    magnitude = "KB";
	}
	if (size > 999) {
	    size = size / 1024;
	    magnitude = "MB";
	}
	if (size > 999) {
	    size = size / 1024;
	    magnitude = "GB";
	}
	if (size > 999) {
	    size = size / 1024;
	    magnitude = "TB";
	}
	size = size.toFixed(1);
	size = addCommas(size);
	size = size + " " + magnitude;
	
	return size;
    }
    
    function addCommas(nStr) {
	nStr += '';
	x = nStr.split('.');
	x1 = x[0];
	x2 = x.length > 1 ? '.' + x[1] : '';
	var rgx = /(\d+)(\d{3})/;
	while (rgx.test(x1)) {
	    x1 = x1.replace(rgx, '$1' + ',' + '$2');
	}
	return x1 + x2;
    }
    
    // executes the callback functions for a given type
    stm.callback = function (type) {
	type = type.toLowerCase();
	for (var c = 0; c < CallbackList[type].length; c++) {
	    CallbackList[type][c][0].call(null, CallbackList[type][c][1], type);
	}
	CallbackList[type] = null;
    };
    
    // deletes an object from the DataStore
    stm.delete_object = function (type, id) {
	    type = type.toLowerCase();
	    if (stm.DataStore[type][id]) {
	        delete stm.DataStore[type][id];
	        stm.TypeData['object_count'][type]--;
	        if (stm.TypeData['object_count'][type] == 0) {
		        delete_object_type(type);
	        }
	    }
    };
    
    // deletes a set of objects from the DataStore
    stm.delete_objects = function (type, ids) {
	    type = type.toLowerCase();
	    for (var i = 0; i < ids.length; i++) {
	        delete_object(type, ids[i]);
	    }
    };
    
    // deletes an entire type from the DataStore
    stm.delete_object_type = function (type) {
	    type = type.toLowerCase();
	    if (stm.TypeData['object_count'][type]) {
	        delete stm.DataStore[type];
	        delete stm.TypeData['object_count'][type];
	        stm.TypeData['type_count']--;
	    }
    };

    // session dumping
    stm.dump = function () {
	var dstring = "";
	dstring += "{";
	for (i in stm.DataStore) {
	    if (stm.DataStore.hasOwnProperty(i)) {
		dstring += '"'+i+'":[';
		for (h in stm.DataStore[i]) {
		    if (stm.DataStore[i].hasOwnProperty(h)) {
			dstring += JSON.stringify(stm.DataStore[i][h]);
			dstring += ",";
		    }
		}
		dstring = dstring.slice(0,-1);
		dstring += "],";
	    }
	}
	dstring = dstring.slice(0,-1);
	dstring += "}";
	stm.saveAs(dstring,"session.dump");
	/*var w = window.open('', '_blank', '');
	w.document.open();
	w.document.write(dstring);
	w.document.close();
	w.document.title = "session.dump";*/
    };

    // save as dialog
    stm.saveAs = function (data, filename) {
	data = 'data:application/octet-stream;base64,' + window.btoa(data);
	var anchor = document.createElement('a');
	anchor.setAttribute('download', filename || "download.txt");
	anchor.setAttribute('href', data);
	document.body.appendChild(anchor);
	anchor.click();
	document.body.removeChild(anchor);
    };

}).call(this);
