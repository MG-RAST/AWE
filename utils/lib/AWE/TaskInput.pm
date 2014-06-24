package AWE::TaskInput;

use strict;
use warnings;

use JSON;
use Data::Dumper;

# leave localfile undef if you do not upload the file
sub new {
    my ($class, %h) = @_;
	
	
	# $localfile, $filename, $host, $node
	
    my $self = {
		data => $h{'data'}, # must be a reference to a scalar
		localfile => $h{'localfile'},
		reference => $h{'reference'}, # specify TaskOutput to use as input
		
		filename => $h{'filename'} || "",
		host => $h{'host'} || "",
		node => $h{'node'} || "",
		origin => $h{'origin'} || "",
	};
	
	if (defined $h{'data'}) {
		if (ref($h{'data'}) ne 'SCALAR') {
			die "error: (TaskInput) data must be reference to scalar, ref=".ref($h{'data'});
		}
	}
	
	
	
	
    bless $self, $class;
	
	$self->reference_update();
	
    return $self;
}

# only used to find upload file
sub localfile {
    my ($self) = @_;
    return $self->{localfile};
}

sub data {
    my ($self) = @_;
    return $self->{data};
}



sub filename {
    my ($self) = @_;
    return $self->{filename};
}

sub reference {
    my ($self) = @_;
    return $self->{reference};
}


sub host {
    my ($self, $host) = @_;
	if (defined $host) {
		$self->{host} = $host;
	}
    return $self->{host};
}
sub node {
    my ($self, $node) = @_;
	if (defined $node) {
		$self->{node} = $node;
	}
    return $self->{node};
}

sub origin {
    my ($self) = @_;
	if (defined $self->reference) {
		my $taskref = $self->reference->taskref || die;
		my $taskid = $taskref->taskid;
		unless (defined $taskid) {
			die "taskid undefined";
		}
		if ($taskid eq '') {
			die "taskid is empty string";
		}
		return $taskref->taskid;
	}
	
    return $self->{origin};
}

sub reference_update {
	my ($self) = @_;
	
	if (defined $self->reference) {
		#print Dumper($self);
		$self->{host}		= $self->reference->host;
		
		
		
		my $filename = $self->reference->filename;
		#print "filename: $filename\n";
		if ($filename =~ /\//) {
			my ($filename_base) = $filename =~ /\/([^\/]*)$/;
			$filename = $filename_base;
		}
		#print "filename: $filename\n";
		#exit(0);
		$self->{filename}	= $filename;
		
		#print Dumper($self->reference);
		#print "--------------------\n";
	}
	
	#exit(0);
}


sub getPair {
	my ($self) = @_;
	
	$self->reference_update();
	
	my $f = {
		'host' => $self->host,
		'node' => $self->node,
		'origin' => $self->origin()
	};
	
	return ($self->filename => $f);

}

1;
