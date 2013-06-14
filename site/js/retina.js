/*
 * retina.js
 *
 */
(function () {
    var root = this;
    var Retina = root.Retina = {};
    var dataServiceURI;
    var renderer_resources  = [];
    var available_renderers = {};
    var loaded_renderers    = {};  
    var widget_resources    = [];
    var available_widgets   = {};
    var loaded_widgets      = {};
    var library_queue       = [];
    var library_resource    = null;
    var loaded_libraries    = {};
    var RendererInstances   = Retina.RendererInstances = [];
    var WidgetInstances     = Retina.WidgetInstances = [];
    
    //
    // initialization
    //
    Retina.init = function (settings) {
	var promise = jQuery.Deferred();
	var promises = [];
	
	var rendererResources = settings.renderer_resources;
	if (rendererResources) {
	    for (i in rendererResources) {
		promises.push(Retina.query_renderer_resource(rendererResources[i]));
	    }
	}
	
	var widgetResources = settings.widget_resources;
	if (widgetResources) {
	    for (i in widgetResources) {
		promises.push(Retina.query_widget_resource(widgetResources[i]));
	    }
	}
		
	var libraryResource = settings.library_resource;
	if (libraryResource) {
	    library_resource = libraryResource;
	    
	    if (libraryResource.match(/^http\:\/\/raw.github.com/)) {
		// adjust bootstrap to the library location
		var cssRuleCode = document.all ? 'rules' : 'cssRules'; // account for IE and FF
		var styles = document.styleSheets;
		for (i=0;i<styles.length;i++) {
	    	    if (styles[i].href && styles[i].href.indexOf('bootstrap.min.css') > -1) {
	    		for (h=0; h<styles[i][cssRuleCode].length; h++) {
	    		    if (styles[i][cssRuleCode][h].selectorText == '[class^="icon-"], [class*=" icon-"]') {
	    			styles[i][cssRuleCode][h].style.backgroundImage = "url('http://raw.github.com/MG-RAST/Retina/master/images/glyphicons-halflings.png')";
	    		    }
	    		    if (styles[i][cssRuleCode][h].selectorText == '.icon-white') {
	    			styles[i][cssRuleCode][h].style.backgroundImage = "url('http://raw.github.com/MG-RAST/Retina/master/images/glyphicons-halflings-white.png')";
	    		    }
	    		}
	    		break;
	    	    }
		}
	    }
	}

	jQuery.when.apply(this, promises).then(function() {
	    promise.resolve();
	});
	
	return promise;
    };
    
    // Utility fuctions
    Retina.each = function (array, func) {
	for (var i = 0; i < array.length; i++) {
	    func(array[i]);
	}
	return array;
    };
    
    Retina.extend = function (object) {
	Retina.each(Array.prototype.slice.apply(arguments), function (source) {
	    for (var property in source) {
		if (!object[property]) {
		    object[property] = source[property];
		}
	    }
	});
	return object;
    };
    
    Retina.keys = function (object) {
	if (object !== Object(object)) throw new TypeError('Invalid object');
	var keys = [];
	for (var key in object) {
	    if (object.hasOwnProperty(key)) {
		keys[keys.length] = key;
	    }
	}
	return keys;
    };
    
    // Retrieve the values of an object's properties.
    Retina.values = function (object) {
	var values = [];
	for (var key in this) {
	    if (object.hasOwnProperty(key)) {
		values[values.length] = object[key];
	    }
	}
	return values;
    };
    
    Retina.require = function (resource, successCb, errorCb) {
	var promise = Retina.load_library(resource);
	promise.then(successCb, errorCb);
	return promise;
    }
     
    Retina.dataURI = function (path) { return dataServiceURI + path; };
    Retina.getJSON = function (path, callback) {
	var url = Retina.dataURI(path);
	jQuery.ajax({
	    url: url,
	    dataType: 'json',
	    data: [],
	    success: callback,
	    error: function (event, request, settings) {
		console.warn("AJAX error! ", event, request, settings);
	    }
	});
    };
    
    Retina.mouseCoords = function (ev) {
	if (ev.pageX || ev.pageY) {
	    return {
		x: ev.pageX,
		y: ev.pageY
	    };
	}
	return {
	    x: ev.clientX + document.body.scrollLeft - document.body.clientLeft,
	    y: ev.clientY + document.body.scrollTop - document.body.clientTop
	};
    };

    Retina.capitalize = function (string) {
	if (string == null || string == "") return string;
	return string[0].toUpperCase() + string.slice(1);
    }

    Retina.wait = function (ms) {
	ms += new Date().getTime();
	while (new Date() < ms){}
    }

    // get date string from ISO8601 timestamp
    Retina.date_string = function (timestamp) {
        var date = new Date(timestamp);
        return date.toLocaleString();
    };
    
    // awsome code i found to produce RFC4122 complient UUID v4
    Retina.uuidv4 = function(a,b) {
        for (b=a=''; a++<36; b+=a*51&52?(a^15?8^Math.random()*(a^20?16:4):4).toString(16):'-');
        return b;
    };

    // find the x and y coordinates of an object on the page
    Retina.findPos = function (obj) {
	var curleft = curtop = 0;
	if (obj.offsetParent) {
	    do {
		curleft += obj.offsetLeft;
		curtop += obj.offsetTop;
	    } while (obj = obj.offsetParent);
	}
	return [curleft,curtop];
    }

    Retina.Numsort = function (a, b) {
	return a - b;
    }

    Retina.Base64 = {
	
	// private property
	_keyStr : "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=",
	
	// public method for encoding
	encode : function (input) {
            var output = "";
            var chr1, chr2, chr3, enc1, enc2, enc3, enc4;
            var i = 0;
	    
            input = Retina.Base64._utf8_encode(input);
	    
            while (i < input.length) {
		
		chr1 = input.charCodeAt(i++);
		chr2 = input.charCodeAt(i++);
		chr3 = input.charCodeAt(i++);
		
		enc1 = chr1 >> 2;
		enc2 = ((chr1 & 3) << 4) | (chr2 >> 4);
		enc3 = ((chr2 & 15) << 2) | (chr3 >> 6);
		enc4 = chr3 & 63;
		
		if (isNaN(chr2)) {
                    enc3 = enc4 = 64;
		} else if (isNaN(chr3)) {
                    enc4 = 64;
		}
		
		output = output +
		    this._keyStr.charAt(enc1) + this._keyStr.charAt(enc2) +
		    this._keyStr.charAt(enc3) + this._keyStr.charAt(enc4);
		
            }
	    
            return output;
	},
	
	// public method for decoding
	decode : function (input) {
            var output = "";
            var chr1, chr2, chr3;
            var enc1, enc2, enc3, enc4;
            var i = 0;
	    
            input = input.replace(/[^A-Za-z0-9\+\/\=]/g, "");
	    
            while (i < input.length) {
		
		enc1 = this._keyStr.indexOf(input.charAt(i++));
		enc2 = this._keyStr.indexOf(input.charAt(i++));
		enc3 = this._keyStr.indexOf(input.charAt(i++));
		enc4 = this._keyStr.indexOf(input.charAt(i++));
		
		chr1 = (enc1 << 2) | (enc2 >> 4);
		chr2 = ((enc2 & 15) << 4) | (enc3 >> 2);
		chr3 = ((enc3 & 3) << 6) | enc4;
		
		output = output + String.fromCharCode(chr1);
		
		if (enc3 != 64) {
                    output = output + String.fromCharCode(chr2);
		}
		if (enc4 != 64) {
                    output = output + String.fromCharCode(chr3);
		}
		
            }
	    
            output = Retina.Base64._utf8_decode(output);
	    
            return output;
	    
	},
	
	// private method for UTF-8 encoding
	_utf8_encode : function (string) {
            string = string.replace(/\r\n/g,"\n");
            var utftext = "";
	    
            for (var n = 0; n < string.length; n++) {
		
		var c = string.charCodeAt(n);
		
		if (c < 128) {
                    utftext += String.fromCharCode(c);
		}
		else if((c > 127) && (c < 2048)) {
                    utftext += String.fromCharCode((c >> 6) | 192);
                    utftext += String.fromCharCode((c & 63) | 128);
		}
		else {
                    utftext += String.fromCharCode((c >> 12) | 224);
                    utftext += String.fromCharCode(((c >> 6) & 63) | 128);
                    utftext += String.fromCharCode((c & 63) | 128);
		}
		
            }
	    
            return utftext;
	},
	
	// private method for UTF-8 decoding
	_utf8_decode : function (utftext) {
            var string = "";
            var i = 0;
            var c = c1 = c2 = 0;
	    
            while ( i < utftext.length ) {
		
		c = utftext.charCodeAt(i);
		
		if (c < 128) {
                    string += String.fromCharCode(c);
                    i++;
		}
		else if((c > 191) && (c < 224)) {
                    c2 = utftext.charCodeAt(i+1);
                    string += String.fromCharCode(((c & 31) << 6) | (c2 & 63));
                    i += 2;
		}
		else {
                    c2 = utftext.charCodeAt(i+1);
                    c3 = utftext.charCodeAt(i+2);
                    string += String.fromCharCode(((c & 15) << 12) | ((c2 & 63) << 6) | (c3 & 63));
                    i += 3;
		}
		
            }
	    
            return string;
	}
	
    }

    Number.prototype.formatString = function(c, d, t) {
	var n = this, c = isNaN(c = Math.abs(c)) ? 0 : c, d = d == undefined ? "." : d, t = t == undefined ? "," : t, s = n < 0 ? "-" : "", i = parseInt(n = Math.abs(+n || 0).toFixed(c)) + "", j = (j = i.length) > 3 ? j % 3 : 0;
	return s + (j ? i.substr(0, j) + t : "") + i.substr(j).replace(/(\d{3})(?=\d)/g, "$1" + t) + (c ? d + Math.abs(n - i).toFixed(c).slice(2) : "");
    };

    /* ===================================================
     * Retina.Widget
     */
    var Widget = Retina.Widget = {};
    Widget.extend = function (spec) {
	var widget = jQuery.extend(true, {}, Widget);
	spec = (spec || {});
	var about;
	switch (typeof spec.about) {
	case 'function' : about = spec.about();  break;
	case 'object'   : about = spec.about;    break;
	default         : about = {};            break;
	};
	
	if (spec.setup && typeof spec.setup !== 'function') {
	    throw "setup() must be a function returning a string.";
	}
	jQuery.extend(widget, spec);
	
	Retina.extend(widget, {
	    loadRenderer: function (args) {
		return Retina.load_renderer(args);
	    },
	    setup: function (args) { return [] },
	    display: function () {},
	    getJSON: Retina.getJSON
	});
	if (widget.about.name) {
	    if (typeof(Retina.WidgetInstances[widget.about.name]) == 'undefined') {
		Retina.WidgetInstances[widget.about.name] = [];
	    }
	    widget.index = Retina.WidgetInstances[widget.about.name].length;
	    Retina.WidgetInstances[widget.about.name].push(widget);
	} else {
	    alert('called invalid renderer, missing about.name');
	    return null;
	}
	return widget;
    };

    Widget.create = function (element, args, nodisplay) {
	var widgetInstance = jQuery.extend(true, {}, Retina.WidgetInstances[element][0]);
	widgetInstance.index = Retina.WidgetInstances[element].length;
	jQuery.extend(true, widgetInstance, args);
	Retina.WidgetInstances[element].push(widgetInstance);

	if (! nodisplay) {
	    widgetInstance.display(args);
	}
	
	return widgetInstance;
    }
    
    /* ===================================================
     * Retina.Renderer
     */
    var Renderer = Retina.Renderer = {};
    Renderer.extend = function (spec) {
	var renderer = jQuery.extend(true, {}, Renderer);
	renderer.settings = {};
	spec = (spec || {});
	jQuery.extend(renderer, spec);
	jQuery.extend(true, renderer.settings, renderer.about.defaults);
	if (renderer.about.name) {
	    if (typeof(Retina.RendererInstances[renderer.about.name]) == 'undefined') {
		Retina.RendererInstances[renderer.about.name] = [];
	    }
	    renderer.index = Retina.RendererInstances[renderer.about.name].length;
	    Retina.RendererInstances[renderer.about.name].push(renderer);
	} else {
	    alert('called invalid renderer, missing about.name');
	    return null;
	}
	
	return renderer;
    };

    Renderer.create = function (rend, settings) {
	var renderer_instance = jQuery.extend(true, {}, Retina.RendererInstances[rend][0]);
	renderer_instance.index = Retina.RendererInstances[rend].length;
	jQuery.extend(true, renderer_instance.settings, settings);
	Retina.RendererInstances[rend].push(renderer_instance);
	return renderer_instance;
    }
    
    //
    // resource section
    //
    Retina.query_renderer_resource = function (resource, list) {
	var promise = jQuery.Deferred();
	
	jQuery.getJSON(resource, function (data) {
	    renderer_resources.push(resource);
	    for (i = 0; i < data.length; i++) {
		var rend = {};
		rend.resource = resource;
		rend.name = data[i].substring(data[i].indexOf(".") + 1, data[i].lastIndexOf("."));
		rend.filename = data[i];
		available_renderers[rend.name] = rend;
	    }
	    if (list) {
		Retina.update_renderer_list(list);
	    }
	    promise.resolve();
	});
	
	return promise;
    };
    
    Retina.update_renderer_list = function (list) {
	var renderer_select = document.getElementById(list);
	if (renderer_select) {
	    renderer_select.options.length = 0;
	    for (i in available_renderers) {
		renderer_select.add(new Option(i, i), null);
	    }
	}
    };
    
    Retina.query_widget_resource = function (resource, list) {
	var promise = jQuery.Deferred();
	
	jQuery.getJSON(resource, function (data) {
	    widget_resources.push(resource);
	    for (ii=0; ii < data.length; ii++) {
		var widget = {};
		widget.resource = resource;
		widget.name = data[ii].substring(data[ii].indexOf(".") + 1, data[ii].lastIndexOf("."));
		widget.filename = data[ii];
		available_widgets[widget.name] = widget;
	    }
	    if (list) {
		Retina.update_widget_list(list);
	    }
	    promise.resolve();
	});
	
	return promise;
    };

    //
    // renderers
    //
    Retina.test_renderer = function (params) {
	if (params.ret) {
	    params.target.innerHTML = "";
	    
	    Retina.Renderer[params.renderer].render({ data: Retina.Renderer[params.renderer].exampleData(), target: params.target });
	} else {
	    params.ret = 1;
	    Retina.load_renderer(params.renderer).then(function() {
		Retina.test_renderer(params);
	    });
	}
    };

    // renderer = { name, resource, filename }
    Retina.add_renderer = function (renderer) {
        if (! available_renderers.hasOwnProperty(renderer.name)) {
	        available_renderers[renderer.name] = renderer;
        }
    }
    
    Retina.load_renderer = function (renderer) {
	var promise;
	if (loaded_renderers[renderer]) {
	    promise = loaded_renderers[renderer];
	} else {
	    promise = jQuery.Deferred();
	    loaded_renderers[renderer] = promise;
	    
	    var promises = [];
	    
	    var rend_data = available_renderers[renderer];
	    var script_url = rend_data.resource + rend_data.filename;
	    jQuery.getScript(script_url).then(function() {
		var requires = Retina.RendererInstances[renderer][0].about.requires;
		for (var i=0; i<requires.length; i++) {
		    promises.push(Retina.load_library(requires[i]));
		}
		
		jQuery.when.apply(this, promises).then(function() {
		    promise.resolve();
		});
	    }, function(jqXHR, textStatus, errorThrown) {
		if (textStatus === 'parsererror') {
		    console.log(errorThrown);
		    parserError(script_url);
		}
	    });
	}

	return promise;
    };
    
    // widget = { name, resource, filename }
    Retina.add_widget = function (widget) {
        if (! available_widgets.hasOwnProperty(widget.name)) {
	        available_widgets[widget.name] = widget;
        }
    }
    
    Retina.load_widget = function (widget) {
	var promise;
	if (loaded_widgets[widget]) {
	    promise = loaded_widgets[widget];
	} else {
	    promise = jQuery.Deferred();
	    loaded_widgets[widget] = promise;
	    
	    var promises = [];
	    
	    var widget_data = available_widgets[widget];
	    var script_url = widget_data.resource + widget_data.filename;
	    jQuery.getScript(script_url).then(function() {
		var requires = Retina.WidgetInstances[widget][0].about.requires;
		for (var i=0; i<requires.length; i++) {
		    promises.push(Retina.load_library(requires[i]));
		}
		var setup = Retina.WidgetInstances[widget][0].setup();
		for (var i=0; i<setup.length; i++) {
		    promises.push(setup[i]);
		}
		
		jQuery.when.apply(this, promises).then(function() {
		    promise.resolve();
		});
	    }, function(jqXHR, textStatus, errorThrown) {
		if (textStatus === 'parsererror') {
		    console.log(errorThrown);
		    parserError(script_url);
		}
	    });
	}
	
	return promise;
    };
    
    Retina.load_library = function (library) {
	var promise;
	if (library == undefined) {
	    library = library_queue[0];
	}
	if (loaded_libraries[library]) {
	    promise = loaded_libraries[library];
	} else {
	    promise = jQuery.Deferred();
	    loaded_libraries[library] = promise;
	    
	    if (library_queue.length) {
		loaded_libraries[library_queue[library_queue.length - 1]].then(Retina.load_library());
		library_queue.push(library);
		return promise;
	    } else {

		var script_url = library_resource + library;
		jQuery.getScript(script_url).then(function() {
		    library_queue.shift();
		    promise.resolve();
		}, function(jqXHR, textStatus, errorThrown) {
		    if (textStatus === 'parsererror') {
			console.log(errorThrown);
			parserError(script_url);
		    }
		});
	    }
	}
	
	return promise;
    };
    
    function parserError(script_url) {
	var error = "ParserError: '" + script_url + "' has a syntax error";
	
	if (jQuery.isFunction(alert)) {
	    alert(error);
	}
	
	throw error;
    }

    /* create an image from an svg  */
    Retina.svg2png = function (source, target, width, height) {
	Retina.load_library('canvg.js').then( function () {
	    var svg = source.innerHTML;
	    svg = svg.replace(/:/, "");
	    svg = svg.replace(/xlink:/g, "");
	    var canvas = document.createElement('canvas');
	    canvas.setAttribute("width", width+"px");
	    canvas.setAttribute("height", height+"px");
	    target.appendChild(canvas);
	    canvg(canvas, svg);
	} );
    }
}).call(this);
