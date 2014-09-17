package AWE::Client;

use strict;
use warnings;
no warnings('once');

use File::Basename;
use Data::Dumper;
use JSON;
use LWP::UserAgent;

1;

our $global_debug = 0;

sub new {
    my ($class, $awe_url, $shocktoken, $awetoken, $debug) = @_;
    
    my $agent = LWP::UserAgent->new;
    my $json = JSON->new;
    $json = $json->utf8();
    $json->max_size(0);
    $json->allow_nonref;
    
	if (defined($shocktoken) && $shocktoken eq '') {
		$shocktoken = undef;
	}
	
	if (defined($awetoken) && $awetoken eq '') {
		$awetoken = undef;
	}
	
	unless (defined $awetoken) {
		$awetoken = $shocktoken; # TODO ugly work around
	}
	
	
    my $self = {
        json => $json,
        agent => $agent,
        awe_url => $awe_url || '',
        shocktoken => $shocktoken,
		awetoken => $awetoken,
        transport_method => 'requests',
		debug => $debug || $global_debug
    };
    if (system("type shock-client > /dev/null 2>&1") == 0) {
        $self->{transport_method} = 'shock-client';
    }

    bless $self, $class;
    return $self;
}

sub debug {
    my ($self) = @_;
    return $self->{debug};
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
sub awetoken {
    my ($self) = @_;
    return $self->{awetoken};
}
sub transport_method {
    my ($self) = @_;
    return $self->{transport_method};
}



sub pretty {
	my ($self, $hash) = @_;
	
	return $self->json->pretty->encode ($hash);
}

# example: getJobQueue('info.clientgroups' => 'yourclientgroup')
sub getJobQueue {
	my ($self, %query) = @_;
	
	$query{'query'}=undef;

	return $self->request('GET', 'job', \%query);
}

sub create_url {
	my ($self, $resource, %query) = @_;
	
	
	unless (defined $self->awe_url ) {
		die "awe_url not defined";
	}
	
	if ($self->awe_url eq '') {
		die "awe_url string empty";
	}
	
	my $my_url = $self->awe_url . "/$resource";
	
	#if (defined $self->token) {
	#	$query{'auth'}=$self->token;
	#}
	
	#build query string:
	my $query_string = "";
	
	foreach my $key (keys %query) {
		my $value = $query{$key};
		
		if ($query_string ne '') {
			$query_string .= '&';
		}
		
		unless (defined $value) {
			$query_string .= $key;
			next;
		}
		
		my @values=();
		if (ref($value) eq 'ARRAY') {
			@values=@$value;
		} else {
			@values=($value);
		}
		
		foreach my $value (@values) {
			if ((length($query_string) != 0)) {
				$query_string .= '&';
			}
			$query_string .= $key.'='.$value;
		}
		
	}
	
	
	if (length($query_string) != 0) {
		
		#print "url: ".$my_url.'?'.$query_string."\n";
		$my_url .= '?'.$query_string;#uri_escape()
	}
	
	
	
	
	return $my_url;
}

sub request {
	#print 'request: '.join(',',@_)."\n";
	my ($self, $method, $resource, $query, $headers) = @_;
	
	
	my $my_url = $self->create_url($resource, (defined($query)?%$query:()));
	
	print "request: $method $my_url\n";
	
	if ($self->{'debug'} ==1) {
		if (defined $self->token) {
			print "found AWE token: ".substr($self->awetoken, 0, 20)."... \n";
		} else {
			print "found no AWE token\n";
		}
		
	}
	
	my @method_args=($my_url, ($self->awetoken)?('Authorization' , "OAuth ".$self->awetoken):());
	
	if (defined $headers) {
		push(@method_args, %$headers);
	}
	
	if ($self->{'debug'} ==1) {
		#print 'method_args: '.join(',', @method_args)."\n";
		print 'method_args: '.Dumper(@method_args)."\n";
	}
	
	#print 'method_args: '.join(',', @method_args)."\n";
	
	my $response_content = undef;
    
    eval {
		
        my $response_object = undef;
		
        if ($method eq 'GET') {
			$response_object = $self->agent->get(@method_args );
		} elsif ($method eq 'DELETE') {
			$response_object = $self->agent->delete(@method_args );
		} elsif ($method eq 'POST') {
			$self->agent->show_progress(1);
			$response_object = $self->agent->post(@method_args );
		} elsif ($method eq 'PUT') {
			$self->agent->show_progress(1);
			$response_object = $self->agent->put(@method_args );
		} else {
			die "method \"$method\"not implemented";
		}
		
		
		$response_content = $self->json->decode( $response_object->content );
        
    };
    
	if ($@ || (! ref($response_content))) {
        print STDERR "[error] unable to connect to AWE ".$self->awe_url."\n";
        return undef;
    } elsif (exists($response_content->{error}) && $response_content->{error}) {
        print STDERR "[error] unable to send $method request to AWE: ".$response_content->{error}[0]."\n";
		return undef;
    } else {
        return $response_content;
    }
	
}

sub deleteJob {
	my ($self, $job_id) = @_;
	
	return $self->request('DELETE', 'job/'.$job_id);
}


sub showJob {
	my ($self, $job_id) = @_;
	
	return $self->request('GET', 'job/'.$job_id);
}

sub resumeJob {
	my ($self, $job_id) = @_;
	
	return $self->request('PUT', 'job/'.$job_id, {'resume' => undef});
}

sub suspendJob {
	my ($self, $job_id) = @_;
	
	return $self->request('PUT', 'job/'.$job_id, {'suspend' => undef});
}

sub resubmitJob {
	my ($self, $job_id) = @_;
	
	return $self->request('PUT', 'job/'.$job_id, {'resubmit' => undef});
}

sub resumeClient {
	my ($self, $client_id) = @_;
	
	return $self->request('PUT', 'client/'.$client_id, {'resume' => undef});
}

sub suspendClient {
	my ($self, $client_id) = @_;
	
	return $self->request('PUT', 'client/'.$client_id, {'suspend' => undef});
}

sub getClientList {
	my ($self) = @_;
	
	
	return $self->request('GET', 'client');

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
	
	
	my $header = {Content_Type => 'multipart/form-data', Content => $content};
	if (defined $self->shocktoken) {
		$header->{Datatoken} = $self->shocktoken;
	}
	
	return $self->request(
		'POST',
		'job',
		undef,
		$header
	);
	
	#my $content = {upload => [undef, "n/a", Content => $awe_qiime_job_json]};
#	my $job_url = $self->awe_url.'/job';
#	
#	my $respond_content=undef;
#	eval {
#        
#		my $http_response = $self->agent->post( $job_url, Datatoken => $self->shocktoken, Content_Type => 'multipart/form-data', Content => $content );
#		$respond_content = $self->json->decode( $http_response->content );
#	};
#	if ($@ || (! ref($respond_content))) {
#        print STDERR "[error] unable to connect to AWE ".$self->awe_url."\n";
#        return undef;
#    } elsif (exists($respond_content->{error}) && $respond_content->{error}) {
#        print STDERR "[error] unable to post data to AWE: ".$respond_content->{error}[0]."\n";
#    } else {
#        return $respond_content;
#    }
}

sub getJobStatus {
	my ($self, $job_id) = @_;
	
	return $self->request('GET', 'job/'.$job_id);
}


sub checkClientGroup {
	my ($self, $clientgroup) = @_;
	
	unless (defined $clientgroup) {
		die "clientgroup undefined"; 
	}
	
	my $client_list_hash = $self->getClientList() || die "client list undefined";
	#print Dumper($client_list_hash);
	
	print "\nOther clients:\n";
	my $found_active_clients = 0;
	my $other_clients = 0;
	foreach my $client ( @{$client_list_hash->{'data'}} ) {
		unless (defined($client->{group}) && ($client->{group} eq $clientgroup)) {
			print $client->{name}." (".$client->{Status}.")  group: ".$client->{group}."  apps: ".join(',',@{$client->{apps}})."\n";
			$other_clients++;
		}
	}
	if ($other_clients == 0) {
		print "none.\n";
	}
	
	print "\nClients in clientgroup \"$clientgroup\":\n";
	foreach my $client ( @{$client_list_hash->{'data'}} ) {
		
		
		
		if (defined($client->{group}) && ($client->{group} eq $clientgroup)) {
			print $client->{name}." (".$client->{Status}.")  group: ".$client->{group}."  apps: ".join(',',@{$client->{apps}})."\n";
			
			if (lc($client->{Status}) eq 'active-idle' || lc($client->{Status}) eq 'active-busy') {
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
	
	print "Summary: found $found_active_clients active client for clientgroup $clientgroup\n";
	return 0;
}

1;

