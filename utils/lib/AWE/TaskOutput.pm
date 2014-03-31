package AWE::TaskOutput;

use strict;
use warnings;

use JSON;



sub new {
    my ($class, $filename, $host, $attrfile) = @_;
	
		
    my $self = {
		filename => $filename || "",
		host => $host || "",
		attrfile => $attrfile || "",
		taskref => undef
	};
	
    bless $self, $class;
    return $self;
}



sub filename {
    my ($self) = @_;
    return $self->{filename};
}
sub host {
    my ($self) = @_;
    return $self->{host};
}
sub attrfile {
    my ($self) = @_;
    return $self->{attrfile};
}

sub taskref {
    my ($self, $task) = @_;
	
	if (defined $task) {
		$self->{taskref} = $task;
	}
	
    return $self->{taskref};
}


sub getPair {
	my ($self) = @_;
	

	
	my $f = {
		'host' => $self->host,
		'attrfile' => $self->attrfile,
	};
	
	return ($self->filename => $f);

}

1;
