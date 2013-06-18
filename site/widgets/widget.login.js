(function () {
    widget = Retina.Widget.extend({
        about: {
            title: "KBase Login",
            name: "login",
            author: "Tobias Paczian",
            requires: [ ]
        }
    });
    
    widget.setup = function () {
	return [];
    };

    widget.callback = null;
    
    widget.display = function (wparams) {
	widget = this;
	var index = widget.index;
	
	// append the modals to the body
	var space = document.createElement('div');
	space.innerHTML = widget.modals(index);
	document.body.appendChild(space);

	// put the login thing into the target space
	wparams.target.innerHTML = widget.login_box(index);

	if (wparams.callback && typeof(wparams.callback) == 'function') {
	    widget.callback = wparams.callback;
	}
    };

    widget.modals = function (index) {
	var html = '\
        <div id="loginModal" class="modal show fade" tabindex="-1" style="width: 400px;" role="dialog" aria-labelledby="loginModalLabel" aria-hidden="true">\
      <div class="modal-header">\
	<button type="button" class="close" data-dismiss="modal" aria-hidden="true">×</button>\
	<h3 id="loginModalLabel">Authenticate to KBase</h3>\
      </div>\
      <div class="modal-body">\
	<p>Enter your KBase credentials.</p>\
        <div id="failure"></div>\
        <table>\
          <tr><th style="vertical-align: top;padding-top: 5px;width: 100px;text-align: left;">login</th><td><input type="text" id="login"></td></tr>\
          <tr><th style="vertical-align: top;padding-top: 5px;width: 100px;text-align: left;">password</th><td><input type="password" id="password"></td></tr>\
        </table>\
      </div>\
      <div class="modal-footer">\
	<button class="btn btn-danger pull-left" data-dismiss="modal" aria-hidden="true">cancel</button>\
	<button class="btn btn-success" onclick="Retina.WidgetInstances.login['+index+'].perform_login('+index+');">log in</button>\
      </div>\
    </div>\
\
    <div id="msgModal" class="modal hide fade" tabindex="-1" style="width: 400px;" role="dialog" aria-labelledby="msgModalLabel" aria-hidden="true">\
      <div class="modal-header">\
	<button type="button" class="close" data-dismiss="modal" aria-hidden="true">×</button>\
	<h3 id="msgModalLabel">Login Information</h3>\
      </div>\
      <div class="modal-body">\
	<p>You have successfully logged in.</p>\
      </div>\
      <div class="modal-footer">\
	<button class="btn btn-success" aria-hidden="true" data-dismiss="modal">OK</button>\
      </div>\
</div>';

	return html;
    };

    widget.login_box = function (index) {
	var html = '\
	<p style="float: right; right:16px; top: 11px; position: relative;cursor: pointer;" onclick="if(document.getElementById(\'login_name\').innerHTML ==\'\'){jQuery(\'#loginModal\').modal(\'show\');}else{if(confirm(\'do you really want to log out?\')){Retina.WidgetInstances.login['+index+'].perform_logout('+index+');}}">\
	        <i class="icon-user icon-white" style="margin-right: 5px;"></i>\
	        <span  id="login_name_span">\
	          <input type="button" class="btn" value="login" style="position:relative; bottom: 2px;" onclick="jQuery(\'#loginModal\').modal(\'show\');">\
	        </span>\
	        <span id="login_name"></span>\
</p>';
	
	return html;
    }
    
    widget.perform_login = function (index) {
	var login = document.getElementById('login').value;
	var pass = document.getElementById('password').value;
	var auth_url = stm.Config.mgrast_api+'?auth='+stm.Config.globus_key+Retina.Base64.encode(login+":"+pass);
	jQuery.get(auth_url, function(d) {
	    if (d && d.token) {
		var uname = d.token.substr(3, d.token.indexOf('|') - 3);
		document.getElementById('login_name_span').style.display = "none";
		document.getElementById('login_name').innerHTML = uname;
		document.getElementById('failure').innerHTML = "";
		stm.Authorization = d.token;
		jQuery('#loginModal').modal('hide');
		jQuery('#msgModal').modal('show');
		if (Retina.WidgetInstances.login[index].callback && typeof(Retina.WidgetInstances.login[index].callback) == 'function') {
		    Retina.WidgetInstances.login[index].callback.call({ 'action': 'login',
									'result': 'success',
									'uname': uname,
									'token': d.token });
		}
	    } else {
		document.getElementById('failure').innerHTML = '<div class="alert alert-error"><button type="button" class="close" data-dismiss="alert">&times;</button><strong>Error:</strong> Login failed.</div>';
		if (Retina.WidgetInstances.login[index].callback && typeof(Retina.WidgetInstances.login[index].callback) == 'function') {
		    Retina.WidgetInstances.login[index].callback.call({ 'action': 'login',
									'result': 'failed',
									'uname': null,
									'token': null });
		}
	    }
	});
    };
    
    widget.perform_logout = function (index) {
	document.getElementById('login_name_span').style.display = "";
	document.getElementById('login_name').innerHTML = "";
	if (Retina.WidgetInstances.login[index].callback && typeof(Retina.WidgetInstances.login[index].callback) == 'function') {
	    Retina.WidgetInstances.login[index].callback.call({ 'action': 'logout'});
	}
    };
    
})();