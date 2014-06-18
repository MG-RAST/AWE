(function () {
    widget = Retina.Widget.extend({
        about: {
            title: "MG-RAST Login",
            name: "login",
            author: "Tobias Paczian",
            requires: [ 'jquery.cookie.js' ]
        }
    });
    
    widget.setup = function () {
	return [];
    };

    widget.callback = null;
    widget.cookiename = "mgauth";
    widget.authResources = RetinaConfig.authResources;
    
    widget.display = function (wparams) {
	widget = this;
	var index = widget.index;
	
	if (wparams && wparams.hasOwnProperty('authResources')) {
	    widget.authResources = wparams.authResources;
	}

	// append the modals to the body
	var space = document.createElement('div');
	space.innerHTML = widget.modals(index);
	document.body.appendChild(space);

	// put the login thing into the target space
	var css = '\
  .userinfo {\
    position: absolute;\
    top: 50px;\
    right: 10px;\
    width: 300px;\
    height: 95px;\
    padding: 10px;\
    background-color: white;\
    color: black;\
    border: 1px solid gray;\
    box-shadow: 4px 4px 4px #666666;\
    border-radius: 6px 6px 6px 6px;\
    z-index: 1000;\
  }\
\
  .userinfo > button {\
    float: right;\
    position: relative;\
    bottom: 0px;\
    margin-left: 10px;\
  }\
\
  .userinfo > img {\
    height: 50px;\
    float: left;\
    margin-right: 10px;\
  }';
	var css_container = document.createElement('style');
	css_container.innerHTML = css;
	document.body.appendChild(css_container);

	widget.target = wparams.target;
	widget.target.innerHTML = widget.login_box(index);

	if (wparams.callback && typeof(wparams.callback) == 'function') {
	    widget.callback = wparams.callback;
	}

	// check for a cookie
	var udata = jQuery.cookie(widget.cookiename);
	if (udata) {
	    udata = JSON.parse(udata);
	    if (udata.hasOwnProperty('user') && udata.user != null) {
		var user = { login: udata.user.login,
			     firstname: udata.user.firstname,
			     lastname: udata.user.lastname,
			     email: udata.user.email,
			   };
		stm.Authentication = udata.token;
		Retina.WidgetInstances.login[index].target.innerHTML = Retina.WidgetInstances.login[index].login_box(index, user);
		if (Retina.WidgetInstances.login[index].callback && typeof(Retina.WidgetInstances.login[index].callback) == 'function') {
		    Retina.WidgetInstances.login[index].callback.call(null, { 'action': 'login',
									      'result': 'success',
									      'token' : udata.token,
									      'user'  : user });
		}
	    }
	}

    };

    widget.modals = function (index) {
	widget = Retina.WidgetInstances.login[index];
	var authResourceSelect = "";
	var loginStyle = "";
	if (Retina.keys(widget.authResources).length > 2) {
	    loginStyle = "class='span3' ";
	    var style = "<style>\
.selector {\
    float: right;\
    border: 1px solid #CCCCCC;\
    border-radius: 4px;\
    padding: 2px;\
    margin-left: 5px;\
    width: 40px;\
}\
.selectImage {\
    width: 21px;\
    margin: 1px;\
    cursor: pointer;\
}\
.hiddenImage {\
    display: none;\
}\
</style>";
	    authResourceSelect = style+"<div class='selector'><i class='icon-chevron-down' style='cursor: pointer;float: right;opacity: 0.2;margin-left: 1px;margin-top: 4px;' onclick=\"if(jQuery('.hiddenImage').css('display')=='none'){jQuery('.hiddenImage').css('display','block');}else{jQuery('.hiddenImage').css('display','none');}\"></i>";
	    for (var i in widget.authResources) {
		if (i=="default") {
		    continue;
		}
		if (i==widget.authResources['default']) {
		    authResourceSelect += "<img class='selectImage' src='images/"+widget.authResources[i].icon+"' onclick=\"Retina.WidgetInstances.login["+index+"].authResources['default']='"+i+"';jQuery('.selectImage').toggleClass('hiddenImage', true);this.className='selectImage';jQuery('.hiddenImage').css('display','none');\">";
		} else {
		    authResourceSelect += "<img class='selectImage hiddenImage' src='images/"+widget.authResources[i].icon+"' onclick=\"Retina.WidgetInstances.login["+index+"].authResources['default']='"+i+"';jQuery('.selectImage').toggleClass('hiddenImage', true);this.className='selectImage';jQuery('.hiddenImage').css('display','none');\">";
		}
	    }
	    authResourceSelect += "</div>";
	    loginStyle = " style='width: 155px;'";
	}
	var html = '\
        <div id="loginModal" class="modal show fade" tabindex="-1" style="width: 400px; display: none;" role="dialog" aria-labelledby="loginModalLabel" aria-hidden="true">\
      <div class="modal-header">\
	<button type="button" class="close" data-dismiss="modal" aria-hidden="true">×</button>\
	<h3 id="loginModalLabel">Authentication</h3>\
      </div>\
      <div class="modal-body">\
	<p>Enter your credentials.</p>\
        <div id="failure"></div>\
        <table>\
          <tr><th style="vertical-align: top;padding-top: 5px;width: 100px;text-align: left;">login</th><td><input type="text" '+loginStyle+'id="login">'+authResourceSelect+'</td></tr>\
          <tr><th style="vertical-align: top;padding-top: 5px;width: 100px;text-align: left;">password</th><td><input type="password" id="password" onkeypress="event = event || window.event;if(event.keyCode == 13) { Retina.WidgetInstances.login['+index+'].perform_login('+index+');}"></td></tr>\
        </table>\
      </div>\
      <div class="modal-footer">\
	<button class="btn btn-danger pull-left" data-dismiss="modal" aria-hidden="true">cancel</button>\
	<button class="btn btn-success" onclick="Retina.WidgetInstances.login['+index+'].perform_login('+index+');">log in</button>\
      </div>\
    </div>\
\
    <div id="msgModal" class="modal hide fade" tabindex="-1" style="width: 400px;" role="dialog" aria-labelledby="msgModalLabel" aria-hidden="true" onkeypress="event = event || window.event;if(event.keyCode == 13) {document.getElementById(\'loginOKButton\').click();}">\
      <div class="modal-header">\
	<button type="button" class="close" data-dismiss="modal" aria-hidden="true">×</button>\
	<h3 id="msgModalLabel">Login Information</h3>\
      </div>\
      <div class="modal-body">\
	<p>You have successfully logged in.</p>\
      </div>\
      <div class="modal-footer">\
	<button class="btn btn-success" aria-hidden="true" data-dismiss="modal" id="loginOKButton">OK</button>\
      </div>\
</div>';

	return html;
    };

    widget.login_box = function (index, user) {
	widget = Retina.WidgetInstances.login[index];

	var html = "";

	var registerEnabled = false;
	var mydataEnabled = false;

	if (user) {
	    html ='\
<div style="float: right; margin-right: 20px; margin-top: 7px; color: gray;">\
   <button class="btn btn-inverse" style="border-radius: 3px 0px 0px 3px; margin-right: -4px;" onclick="if(document.getElementById(\'userinfo\').style.display==\'none\'){document.getElementById(\'userinfo\').style.display=\'\';}else{document.getElementById(\'userinfo\').style.display=\'none\';}">\
      <i class="icon-user icon-white" style="margin-right: 5px;"></i>\
      '+user.firstname+' '+user.lastname+'\
      <span class="caret" style="margin-left: 5px;"></span>\
   </button>\
   <button class="btn btn-inverse" style="border-radius: 0px 3px 3px 0px;">?</button>\
</div>\
<div class="userinfo" id="userinfo" style="display: none;">\
   <img src="images/user.png">\
   <h4 style="margin-top: 5px;">'+user.firstname+' '+user.lastname+'</h4>\
<p style="margin-top: -10px;">'+user.email+'</p>\
   <button class="btn btn-inverse" onclick="document.getElementById(\'userinfo\').style.display=\'none\';Retina.WidgetInstances.login['+index+'].perform_logout('+index+');">logout</button>\
'+(mydataEnabled ? '<button class="btn" style="float: left;">myData</button>' : '')+'\
</div>';
	} else {
	    html ='\
<div style="float: right; margin-right: 20px; margin-top: 7px; color: gray;">\
   <button class="btn btn-inverse" style="border-radius: 3px 0px 0px 3px; margin-right: -4px;" onclick="jQuery(\'#loginModal\').modal(\'show\');document.getElementById(\'login\').focus();">\
      Login\
   </button>\
' + (registerEnabled ? '<button class="btn btn-inverse" style="border-radius: 3px 0px 0px 3px; margin-right: -4px;" onclick="alert(\'register\');">\
      Register\
</button>' : '') +'\
   <button class="btn btn-inverse" style="border-radius: 0px 3px 3px 0px;">?</button>\
</div>';
	}
	
	return html;
    }

    widget.perform_login = function (index) {
	widget = Retina.WidgetInstances.login[index];
	var login = document.getElementById('login').value;
	var pass = document.getElementById('password').value;
	var auth_url = stm.Config.mgrast_api+'?auth='+widget.authResources[widget.authResources.default].prefix+Retina.Base64.encode(login+":"+pass);
	jQuery.get(auth_url, function(d) {
	    if (d && d.token) {
		var user;
		if (d.hasOwnProperty('firstname')) {
		    user = { login: d.login,
			     firstname: d.firstname,
			     lastname: d.lastname,
			     email: d.email,
			   };
		} else {
		    jQuery.ajax({ url: stm.Config.mgrast_api+"/user/"+login,
				  headers: { "Auth": d.token },
				  success: function(result) {
				      user = { login: result.login,
					       firstname: result.firstname,
					       lastname: result.lastname,
					       email: result.email,
					     };
				  },
				  async: false
				});
		}
		stm.Authentication = d.token;
		Retina.WidgetInstances.login[index].target.innerHTML = Retina.WidgetInstances.login[index].login_box(index, user);
		document.getElementById('failure').innerHTML = "";
		stm.Authentication = d.token;
		jQuery('#loginModal').modal('hide');
		jQuery('#msgModal').modal('show');
		jQuery.cookie(Retina.WidgetInstances.login[1].cookiename, JSON.stringify({ "user": user,
											   "token": d.token }), { expires: 7 });
		if (Retina.WidgetInstances.login[index].callback && typeof(Retina.WidgetInstances.login[index].callback) == 'function') {
		    Retina.WidgetInstances.login[index].callback.call(null, { 'action': 'login',
									      'result': 'success',
									      'token' : d.token,
									      'user'  : user  });
		}
	    } else {
		document.getElementById('failure').innerHTML = '<div class="alert alert-error"><button type="button" class="close" data-dismiss="alert">&times;</button><strong>Error:</strong> Login failed.</div>';
		if (Retina.WidgetInstances.login[index].callback && typeof(Retina.WidgetInstances.login[index].callback) == 'function') {
		    Retina.WidgetInstances.login[index].callback.call(null, { 'action': 'login',
									      'result': 'failed',
									      'token' : null,
									      'user'  : null  });
		}
	    }
	}).fail(function() {
	    document.getElementById('failure').innerHTML = '<div class="alert alert-error"><button type="button" class="close" data-dismiss="alert">&times;</button><strong>Error:</strong> Login failed.</div>';
		if (Retina.WidgetInstances.login[index].callback && typeof(Retina.WidgetInstances.login[index].callback) == 'function') {
		    Retina.WidgetInstances.login[index].callback.call(null, { 'action': 'login',
									      'result': 'failed',
									      'token': null,
									      'user': null });
		}
	});
    };
    
    widget.perform_logout = function (index) {
	Retina.WidgetInstances.login[index].target.innerHTML = Retina.WidgetInstances.login[index].login_box(index);
	stm.Authentication = null;
	jQuery.cookie(Retina.WidgetInstances.login[1].cookiename, JSON.stringify({ "token": null,
										   "user": null }));
	if (Retina.WidgetInstances.login[index].callback && typeof(Retina.WidgetInstances.login[index].callback) == 'function') {
	    Retina.WidgetInstances.login[index].callback.call(null, { 'action': 'logout' });
	}
    };
    
})();