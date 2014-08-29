package AWE::Task;

# this is a generic representation of an AWE task. It also serves as parent class for AWE apps.

use strict;
use warnings;



use JSON;



sub new {
    my ($class) = @_;
    
	my $cmd = {
		"name"			=> "",
		"args"			=> "",
		"environ"		=> {},
		"description"	=> ""
	};
			
    my $self = {
		cmd => $cmd,
		inputs => [],
		outputs => [],
		partinfo => {},
		taskid => ""
	};
	
    bless $self, $class;
    return $self;
}

sub cmd {
    my ($self) = @_;
    return $self->{cmd};
}

sub inputs {
    my ($self) = @_;
    return $self->{inputs};
}
sub outputs {
    my ($self) = @_;
    return $self->{outputs};
}
sub partinfo {
    my ($self) = @_;
    return $self->{partinfo};
}
sub taskid {
    my ($self, $newid) = @_;
	
	if (defined $newid) {
		$self->{taskid} = $newid;
	}
	
    return $self->{taskid};
}


sub description {
    my ($self, $newdescription) = @_;
	
	if (defined $newdescription) {
		$self->{'cmd'}->{'description'} = $newdescription;
	}
	
    return $self->{'cmd'}->{'description'};
}

sub environ {
    my ($self, $newenviron) = @_;
	
	if (defined $newenviron) {
		$self->{'cmd'}->{'environ'} = $newenviron;
	}
	
    return $self->{'cmd'}->{'environ'};
}



sub command {
    my ($self, $cmd) = @_;
	
	my ($name, $args) = $cmd =~ /^(\S+)\s+(.*)$/;
	
	unless (defined $name) {
		die "name not found in cmd \"$cmd\"";
	}
	
	$self->{cmd}->{'name'} = $name;
	
	$self->{cmd}->{'args'} = $args || "";
    
}


sub addInput {
	my ($self) = shift(@_);
	
	push(@{$self->inputs}, @_);
}

# returns reference to the output objetc
sub addOutput {
	my ($self, $taskoutput) = @_;
	
	push(@{$self->outputs}, $taskoutput);
	
	#tell output which task it belongs to:
	$taskoutput->taskref($self);
	
	return $taskoutput;
}


sub getHash {
	my ($self) = @_;
	

	my $deps={};
	my $inputs = {};
	
	foreach my $i (@{$self->inputs}) {
		my ($f, $n) = $i->getPair();
		if (defined $inputs->{$f}) {
				die "input node for file $f already exists";
		}
		$inputs->{$f} = $n;
		if (defined $n->{'origin'} && $n->{'origin'} ne '') {
			$deps->{$n->{'origin'}} = 1;
		}
	}
	
	my $outputs = {};
	foreach my $i (@{$self->outputs}) {
		my ($f, $n) = $i->getPair();
		if (defined $outputs->{$f}) {
			die "output node for file $f already exists";
		}
		$outputs->{$f} = $n;
	}
	
	my @dependsOn = keys($deps);
	
	my $t = {
		'cmd' => $self->cmd,
		'dependsOn' => \@dependsOn,
		'inputs' => $inputs,
		'outputs' => $outputs,
		'partinfo' => $self->partinfo,
		'taskid' => $self->taskid
	};
	
	return $t;

}

1;
