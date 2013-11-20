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


