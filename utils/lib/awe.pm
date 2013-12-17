package AWE;

use strict;
use warnings;
no warnings('once');

use File::Basename;
use Data::Dumper;
use JSON;
use LWP::UserAgent;

1;

sub new {
    my ($class, $awe_url, $shocktoken) = @_;
    
    my $agent = LWP::UserAgent->new;
    my $json = JSON->new;
    $json = $json->utf8();
    $json->max_size(0);
    $json->allow_nonref;
    
    my $self = {
        json => $json,
        agent => $agent,
        awe_url => $awe_url || '',
        shocktoken => $shocktoken || '',
        transport_method => 'requests'
    };
    if (system("type shock-client > /dev/null 2>&1") == 0) {
        $self->{transport_method} = 'shock-client';
    }

    bless $self, $class;
    return $self;
}

sub json {
    my ($self) = @_;
    return $self->{json};
}
sub agent {
    my ($self) = @_;
    return $self->{agent};
}
sub awe_url {
    my ($self) = @_;
    return $self->{awe_url};
}
sub shocktoken {
    my ($self) = @_;
    return $self->{shocktoken};
}
sub transport_method {
    my ($self) = @_;
    return $self->{transport_method};
}

# example: getJobQueue('info.clientgroups' => 'yourclientgroup')
sub getJobQueue {
	my ($self) = shift;
	my @pairs = @_;
	my $client_url = $self->awe_url.'/job';
	
	#print "got: ".join(',', @pairs)."\n";
	
	if (@pairs > 0) {
		$client_url .= '?query';
		for (my $i = 0 ; $i < @pairs ; $i+=2) {
			$client_url .= '&'.$pairs[$i].'='.$pairs[$i+1];
		}
	}
	
	my $http_response = $self->agent->get($client_url);
	my $http_response_content  = $http_response->decoded_content;
	die "Can't get url $client_url ", $http_response->status_line
	unless $http_response->is_success;
	
	my $http_response_content_hash = $self->json->decode($http_response_content);
	
	return $http_response_content_hash;
}

sub deleteJob {
	my ($self, $job_id) = @_;
	
	my $deletejob_url = $self->awe_url.'/job/'.$job_id;
	
	my $respond_content=undef;
	eval {
        
		my $http_response = $self->agent->delete( $deletejob_url );
		$respond_content = $self->json->decode( $http_response->content );
	};
	if ($@ || (! ref($respond_content))) {
        print STDERR "[error] unable to connect to AWE ".$self->awe_url."\n";
        return undef;
    } elsif (exists($respond_content->{error}) && $respond_content->{error}) {
        print STDERR "[error] unable to delete job from AWE: ".$respond_content->{error}[0]."\n";
    } else {
        return $respond_content;
    }
}

sub getClientList {
	my ($self) = @_;
	
	unless (defined $self->awe_url) {
		die;
	}
	
	my $client_url = $self->awe_url.'/client/';
	my $http_response = $self->agent->get($client_url);
	
	my $http_response_content  = $http_response->decoded_content;
	#print "http_response_content: ".$http_response_content."\n";
	
	die "Can't get url $client_url ", $http_response->status_line
	unless $http_response->is_success;
	

	my $http_response_content_hash = $self->json->decode($http_response_content);

	return $http_response_content_hash;
}

# submit json_file or json_data
sub submit_job {
	my ($self, %hash) = @_;
	
	 my $content = {};
	if (defined $hash{json_file}) {
		unless (-s $hash{json_file}) {
			die "file not found";
		}
		$content->{upload} = [$hash{json_file}]
	}
	if (defined $hash{json_data}) {
		#print "upload: ".$hash{json_data}."\n";
		$content->{upload} = [undef, "n/a", Content => $hash{json_data}]
	}
	
	#my $content = {upload => [undef, "n/a", Content => $awe_qiime_job_json]};
	my $job_url = $self->awe_url.'/job';
	
	my $respond_content=undef;
	eval {
        
		my $http_response = $self->agent->post( $job_url, Datatoken => $self->shocktoken, Content_Type => 'multipart/form-data', Content => $content );
		$respond_content = $self->json->decode( $http_response->content );
	};
	if ($@ || (! ref($respond_content))) {
        print STDERR "[error] unable to connect to AWE ".$self->awe_url."\n";
        return undef;
    } elsif (exists($respond_content->{error}) && $respond_content->{error}) {
        print STDERR "[error] unable to post data to AWE: ".$respond_content->{error}[0]."\n";
    } else {
        return $respond_content;
    }
}

sub getJobStatus {
	my ($self, $job_id) = @_;
	
	
	my $jobstatus_url = $self->awe_url.'/job/'.$job_id;
	
	my $respond_content=undef;
	eval {
        
		my $http_response = $self->agent->get( $jobstatus_url );
		$respond_content = $self->json->decode( $http_response->content );
	};
	if ($@ || (! ref($respond_content))) {
        print STDERR "[error] unable to connect to AWE ".$self->awe_url."\n";
        return undef;
    } elsif (exists($respond_content->{error}) && $respond_content->{error}) {
        print STDERR "[error] unable to get data from AWE: ".$respond_content->{error}[0]."\n";
    } else {
        return $respond_content;
    }
}


sub checkClientGroup {
	my ($self, $clientgroup) = @_;
	
	my $client_list_hash = $self->getClientList();
	#print Dumper($client_list_hash);
	
	print "\nList of clients:\n";
	my $found_active_clients = 0;
	
	foreach my $client ( @{$client_list_hash->{'data'}} ) {
		
		
		
		unless (defined($client->{group}) && ($client->{group} eq $clientgroup)) {
			print $client->{name}." (".$client->{Status}.")  group: ".$client->{group}."  apps: ".join(',',@{$client->{apps}})."\n";
		}
	}
	print "My clientgroup:\n";
	foreach my $client ( @{$client_list_hash->{'data'}} ) {
		
		
		
		if (defined($client->{group}) && ($client->{group} eq $clientgroup)) {
			print $client->{name}." (".$client->{Status}.")  group: ".$client->{group}."  apps: ".join(',',@{$client->{apps}})."\n";
			
			if (lc($client->{Status}) eq 'active') {
				$found_active_clients++;
			} else {
				print "warning: client not active:\n";
			}
		}
	}
	
	
	if ($found_active_clients == 0) {
		print STDERR "warning: did not find any active client for clientgroup $clientgroup\n";
		return 1;
	}
	
	print "found $found_active_clients active client for clientgroup $clientgroup\n";
	return 0;
}

1;

##########################################
package AWE::Job;
use Data::Dumper;
use Storable qw(dclone);

1;


sub new {
    my ($class, %h) = @_;
	
	my $self = {
		
		'data' => {'info' => $h{'info'}},
		'tasks' => $h{'tasks'},
		
		
		#'trojan' => $h{'trojan'},
		'shockhost' => $h{'shockhost'},
		'task_templates' => $h{'task_templates'}
	};
	
	
		
	
	
	#assignTasks($self, %{$h{'job_input'}});
	assignTasks($self);
	replace_taskids($self);
	#delete $self->{'trojan'};
	#delete $self->{'shockhost'};
	#delete $self->{'task_templates'};
		
	
	bless $self, $class;
    return $self;
}

sub create_simple_template {
	my $cmd = shift(@_);
	#get outputs
	
	#print "cmd1: $cmd\n";
	my @outputs = $cmd =~ /@@(\S+)/g;
	$cmd =~ s/\@\@(\S+)/$1/;
	
	my @inputs = $cmd =~ /[^@]@(\S+)/g;

	#print "outputs: ".join(' ', @outputs)."\n";
	#print "inputs: ".join(' ', @inputs)."\n";
	#print "cmd2: $cmd\n";
	
	my $meta_template = {
		"cmd" => $cmd,
		"inputs" => \@inputs,
		"outputs" => \@outputs,
		"trojan" => {} # creates trojan script with basic features
	};
	
	
	return $meta_template;
}

sub createTask {
	my ($self, %h)= @_;
	
	
	my $taskid = $h{'task_id'};
	my $task_templates = $self->{'task_templates'};
	my $task_template_name = $h{'task_template'};
	my $task_cmd = $h{'task_cmd'};
	
	
	
	
	
	my $task_template=undef;
	my $task;
	
	if (defined $task_template_name) {
		
		
		$task_template = dclone($task_templates->{$task_template_name});
		
	} elsif (defined $task_cmd) {
		$task_template = create_simple_template($task_cmd);
		#print Dumper($task)."\n";
	} else {
		print Dumper(%h)."\n";
		die "no task template found (task_template or task_cmd)";
	}
	
	#print "template:\n";
	#print Dumper($task_template)."\n";
	
	my $host = $h{'shockhost'} || $self->{'shockhost'};
	
	
	
	my $cmd = $task_template->{'cmd'};
	$task->{'cmd'} = undef;
	
	$task->{'cmd'}->{'description'} = $taskid ||  "description";   # since AWE does not accept string taskids, I move taskid into the description
	
	$task->{'taskid'} = $taskid; # will be replaced later by numeric taskid
	
	
	
	if (defined $h{'TROJAN'}) {
		push(@{$task_template->{'inputs'}}, '[TROJAN]');
		#print "use trojan XXXXXXXXXXXXXXXXX\n";
	}
	
	
	
	my $depends = {};
	
	my $inputs = {};
	foreach my $key_io (@{$task_template->{'inputs'}}) {
		
		my ($key) = $key_io =~ /^\[(.*)\]$/;
	
		unless (defined $key) {
			
			die "no input key found in: $key_io";
			next;
		}
		
		my $value = $h{$key};
		if (defined $value) {
			if (ref($value) eq 'ARRAY') {
				my ($source_type, $source, $filename) = @{$value};
				
				if ($source_type eq 'shock') {
					$inputs->{$filename}->{'node'} = $source;
					
				} elsif ($source_type eq 'task') {
					$inputs->{$filename}->{'origin'} = $source;
					$depends->{$source} = 1;
				} else {
					die "source_type $source_type unknown";
				}
				
				
				$inputs->{$filename}->{'host'} = $host;
				$cmd =~ s/\[$key\]/$filename/g;
			} else {
				die "array ref expected for key $key";
			}
			
			
			
		} else {
			die "input key \"$key\" not defined";
		}
	}
	$task->{'inputs'}=$inputs;
	
	
	my $outputs = {};
	foreach my $key_io (@{$task_template->{'outputs'}}) {
		
		
		my ($key) = $key_io =~ /^\[(.*)\]$/;
		
		
		
		unless (defined $key) {
			$outputs->{$key_io}->{'host'} = $host;
			next;
		}
		my $value = $h{$key};
		if (defined $value) {
			$outputs->{$value}->{'host'} = $host;
			$cmd =~ s/\[$key\]/$value/g;
		} else {
			die "output key \"$key\" not defined";
		}
	
	}
	$task->{'outputs'}=$outputs;
	
	$task->{'cmd'}->{'args'} = $cmd;
	
	
	my @depends_on = ();
	foreach my $dep (keys %$depends) {
		push(@depends_on, $dep);
	}
	if (@depends_on > 0 ) {
		$task->{'dependsOn'} = \@depends_on;
	}
	
	#print "task:\n";
	#print Dumper($task)."\n";
#	exit(0);
	
	return $task;
}



sub assignTasks {
	my ($self) = @_;
	
	
	
	my $task_specs = $self->{'tasks'};
	
	
	#replace variables in tasks
	for (my $i =0  ; $i < @{$task_specs} ; ++$i) {
		my $task_spec = $task_specs->[$i];
		
		#print "ref: ".ref($task)."\n";
		#print "keys: ".join(',',keys(%$tasks))."\n";
		#print Data::Dumper($task);
		
		print "task_spec:\n";
		print Dumper($task_spec);
		#my $trojan_file=$task->{'trojan_file'};
		
		

		### createTask ###
		my $newtask = createTask($self, %$task_spec);
		$self->{'data'}->{'tasks'}->[$i] = $newtask;
		
		#if (defined($task_spec->{'TROJAN'})) {
		#	$newtask->{'trojan_file'} = ${$task_spec->{'TROJAN'}}[2];
		#};
		
		#$task = $tasks->[$i];
		
		#print Data::Dumper($task);
		#exit(0);
		
		#my $trojan = $task->{'trojan'};
		
		#my $inputs = $task->{'inputs'};
		
	}
}

sub assignInput {
	my ($self, %h) = @_;
	# $h contains named_input to shock node mapping
	my $tasks = $self->{'data'}->{'tasks'};
	
	
	for (my $i =0  ; $i < @{$tasks} ; ++$i) {
		my $task = $tasks->[$i];
	
		my $task_spec = $self->{'tasks'}->[$i];
		
		
		#my $trojan_file=$task->{'trojan_file'};
		my $trojan_file=undef;
		if (defined($task_spec->{'TROJAN'})) {
			$trojan_file = ${$task_spec->{'TROJAN'}}[2];
		} else {
			die;
		}
	
		#print Dumper($task);
		my $inputs = $task->{'inputs'};
		
		foreach my $inputfile (keys(%{$inputs})) {
			#print "inputfile: $inputfile\n";
			my $input_obj = $inputs->{$inputfile};
			
			
			if (defined $input_obj->{'node'}) {

				my ($variable) = $input_obj->{'node'} =~ /\[(.*)\]/;
				
				if (defined $variable) {
					my $file_obj = $h{$variable};
					if (defined($file_obj)) {
						$input_obj->{'node'} = $file_obj->{'node'};
						$input_obj->{'host'} = $file_obj->{'shockhost'};
					} else {
						die "no replacement for variable $variable found";
					}
				}
				
			
			}
		}
		#print Dumper($task);
		#exit(0);
		#print "got: ".$task->{'cmd'}->{'args'}."\n";
		
		unless (defined $task->{'cmd'}) {
			die;
		}
		
		unless (defined $task->{'cmd'}->{'args'}) {
			die;
		}
		
		
		
		if (defined($trojan_file)) {
		#if ( defined($h{'TROJAN'}) ) {
			
			# modify AWE task to use trojan script
			
			
			
			$task->{'cmd'}->{'args'} = "\@".$trojan_file." ".$task->{'cmd'}->{'args'};
			$task->{'cmd'}->{'name'} = "perl";
			
			
		} else {
			# extract the executable from command
		
			my $executable;
			$task->{'cmd'}->{'args'} =~ s/^([\S]+)//;
			$executable=$1;
			$task->{'cmd'}->{'args'} =~ s/^[\s]*//;
			
			unless (defined $executable) {
				die "executable not found in ".$task->{'cmd'}->{'args'};
			}
			
			$task->{'cmd'}->{'name'} = $executable;
		}
	}
	
}

sub replace_taskids {
	my ($self) = @_;
	
	# $h contains named_input to shock node mapping
	my $tasks = $self->{'data'}->{'tasks'};
	my $taskid_num = {};
	my $taskid_count = 0;
	
	#replace taskids with strings of numbers !
	for (my $i =0  ; $i < @{$tasks} ; ++$i) {
		my $task = $tasks->[$i];
		my $taskid = $task->{'taskid'};
		
		unless (defined $taskid_num->{$taskid}) {
			$taskid_num->{$taskid} = $taskid_count;
			$taskid_count++;
		}
		$task->{'taskid'}= $taskid_num->{$taskid}.'';
		
		
		my $dependsOn = $task->{'dependsOn'};
		if (defined $dependsOn) {
			my $dependsOn_new = [];
			foreach my $dep_task (@$dependsOn) {
				unless (defined $taskid_num->{$dep_task}) {
					$taskid_num->{$dep_task} = $taskid_count;
					$taskid_count++;
				}
				
				push(@$dependsOn_new, $taskid_num->{$dep_task}.'');
			}
			if (@$dependsOn_new > 0) {
				$task->{'dependsOn'} = $dependsOn_new;
			}
		}
		
		my $inputs = $task->{'inputs'};
		
		
		foreach my $input (keys(%$inputs)) {
			
			if (defined($inputs->{$input}->{'origin'})) {
				my $origin = $inputs->{$input}->{'origin'};
				unless (defined $taskid_num->{$origin}) {
					$taskid_num->{$origin} = $taskid_count;
					$taskid_count++;
				}
				$inputs->{$input}->{'origin'} = $taskid_num->{$origin}.'';
			}
		}
		
		
		
	}
	
}

sub hash {
	my ($self) = @_;
	#return {%$self}
	return $self->{'data'};
}

sub json {
	my ($self) = @_;
	my $job_hash = hash($self);
	
	my $json = JSON->new;
	my $job_json = $json->encode( $job_hash );
	return $job_json;
}

############################################
#the trojan horse generator
# - creates log files (stdin, stderr)
# - ENV support
# - archives directories
# - scripts on VM do not need to be registered at AWE client
sub get_trojanhorse {
	my %h = @_;
	
	print "get_trojanhorse got: ".join(' ' , keys(%h))."\n";
	#exit(0);
	
	my $out_dirs = $h{'out_dirs'};
	my $resulttarfile = $h{'tar'};
	
	my $out_files = $h{'out_files'};
	
	
	# start of trojanhorse_script
	my $trojanhorse_script = <<'EOF';
	#!/usr/bin/env perl
	
	use strict;
	use warnings;
	
	use IO::Handle;
	
	open OUTPUT, '>', "trojan.stdout" or die $!;
	open ERROR,  '>', "trojan.stderr"  or die $!;
	
	STDOUT->fdopen( \*OUTPUT, 'w' ) or die $!;
	STDERR->fdopen( \*ERROR,  'w' ) or die $!;
	
	sub systemp {
		print "cmd: ".join(' ', @_)."\n";
		return system(@_);
	}
	
	
	eval {
		print "hello trojan world\n";
		
		my $line = join(' ',@ARGV)."\n";
		print "got: ".$line;
		
		
		my $check = join('|', keys %ENV);
		$line =~ s/\$($check)/$ENV{$1}/g;
		
		systemp($line)==0 or die;
		
	};
	if ($@) {
		print "well.. there was an error, but I catched it.. ;-)\n";
	}
	###TAR###
	
	###OUTFILES###
	
	close(OUTPUT);
	close(ERROR);
	select STDOUT; # back to normal
	
EOF
	# end of trojanhorse_script
	
	if (defined $out_dirs && @$out_dirs > 0) {
		my $outdirsstr = join(' ', @$out_dirs);
		
		# start of tarcmd
		my $tarcmd = <<EOF;
		systemp("tar --ignore-failed-read -cf $resulttarfile $outdirsstr trojan.stdout trojan.stderr")==0 or die;
EOF
		# end of tarcmd
		
		$trojanhorse_script =~ s/\#\#\#TAR\#\#\#/$tarcmd/g;
	}
	
	if (defined $out_files && @$out_files > 0) {
		my $out_files_str = join(' ', @$out_files);
		
		# start of tarcmd
		my $cpcmd = <<EOF;
		systemp("cp $out_files_str .")==0 or die;
EOF
		# end of tarcmd
		
		$trojanhorse_script =~ s/\#\#\#OUTFILES\#\#\#/$cpcmd/g;
	}
	
	
	return $trojanhorse_script;
}
# end of trojan generator
############################################

1;