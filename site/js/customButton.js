function deleteMGRASTJob (job) {
    var widget = Retina.WidgetInstances.awe_monitor[1];
    var reason = prompt('Really delete this job from MG-RAST? This cannot be undone!', "reason for deletion");
    if (reason) {
	var auth = widget.authHeader.Authorization;
	auth = auth.replace('OAuth', 'mgrast');
	jQuery.ajax({
	    method: "POST",
	    dataType: "json",
	    data: '{ "metagenome_id": "'+job.info.userattr.id+'","reason": "'+reason+'" }',
	    processData: false,
	    headers: { "Authorization": auth }, 
	    url: RetinaConfig["mgrast_api"]+"/job/delete",
	    success: function (data) {
		Retina.WidgetInstances.awe_monitor[1].display();
		alert('job deleted from MG-RAST');
	    }}).fail(function(xhr, error) {
		alert('failed to delete job from MG-RAST');
	    });
    }
};

function showDeleteMGRASTButton (job) {
    if (job.state == "suspend" && job.info.pipeline == "mgrast-prod") {
	return true;
    } else {
	return false;
    }
};