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
	my $client_url = $self->awe_url.'/client/';
	my $http_response = $self->agent->get($client_url);
	
	my $http_response_content  = $http_response->decoded_content;
	#print "http_response_content: ".$http_response_content."\n";
	
	die "Can't get url $client_url ", $http_response->status_line
	unless $http_response->is_success;
	

	my $http_response_content_hash = $self->json->decode($http_response_content);

	return $http_response_content_hash;
}


sub upload {
	my ($self, %hash) = @_;
	
	 my $content = {};
	if (defined $hash{json_file}) {
		unless (-s $hash{json_file}) {
			die "file not found";
		}
		$content->{upload} = [$hash{json_file}]
	}
	if (defined $hash{json_data}) {
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

1;

##########################################
package AWE::Job;

1;

sub createTask {
	
	my %h = @_;
	
	my $taskid = $h{'task_id'};
	my $task = $h{'task_tmpl'}; # TODO: make copy of template !
	my $host = $h{'shockhost'};
	
	
	
	my $cmd = $task->{'cmd'};
	$task->{'cmd'} = undef;
	
	$task->{'cmd'}->{'description'} = "description";
	$task->{'cmd'}->{'name'} = "perl";
	
	$task->{'taskid'} = $taskid;
	
	
	my $depends = {};
	
	my $inputs = {};
	foreach my $key (@{$task->{'inputs'}}) {
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
			die "input $key not defined";
		}
	}
	$task->{'inputs'}=$inputs;
	
	
	my $outputs = {};
	foreach my $key (@{$task->{'outputs'}}) {
		my $value = $h{$key};
		if (defined $value) {
			$outputs->{$value}->{'host'} = $host;
			$cmd =~ s/\[$key\]/$value/g;
		} else {
			die "output $key not defined";
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
	
	return $task;
}

sub new {
    my ($class, %h) = @_;
	
	my $self = {
		'tasks' => $h{'tasks'},
		'info' => $h{'info'}
	};
	
	delete $h{'tasks'};
	delete $h{'info'};
	
	
	if (defined $self->{tasks}) {
		assignTasks($self, %h);
	}
	
	bless $self, $class;
    return $self;
}

sub assignTasks {
	my ($self, %h) = @_;
	
	
	
	my $tasks = $self->{'tasks'};
	
	foreach my $task (@{$tasks}) {
		
		#print Dumper($task)."\n";
		my $inputs = $task->{'inputs'};
		
		foreach my $inputfile (keys(%{$inputs})) {
			
			my $input_obj = $inputs->{$inputfile};
			#print Dumper($input_obj)."\n";
			
			if (defined $input_obj->{'node'}) {
				
				#print "found ".$input_obj->{'node'}."\n";
				
				my ($variable) = $input_obj->{'node'} =~ /\[(.*)\]/;
				
				if (defined $variable) {
					my $replacement = $h{$variable};
					if (defined($replacement)) {
						
						$input_obj->{'node'} =~ s/\[$variable\]/$replacement/g;
					} else {
						die "no replacement for variable $variable found";
					}
				}
				
				#foreach my $key (keys(%h)) {
				#	my $value = $h{$key};
				#	$input_obj->{'node'} =~ s/\[$key\]/$value/g;
				#}
			}
		}
		
		
		#if ($trojan == 1) {
	#		my $cmd = $task->{'cmd'};
	#
	#	}
		
		
	}
	
	
	
	
	## compose AWE job ##
	#my $awe_qiime_job = {};
	#$awe_qiime_job->{'tasks'} = $tasks;
	#$awe_qiime_job->{'info'} = $info;
	
	#return $awe_qiime_job;
}

sub hash {
	my ($self) = @_;
	return {%$self}
}

sub json {
	my ($self) = @_;
	my $job_hash = hash($self);
	
	my $json = JSON->new;
	my $job_json = $json->encode( $job_hash );
	return $job_json;
}


1;