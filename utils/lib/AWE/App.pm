package AWE::App;

use strict;
use warnings;



sub get_app_mode {
	my ($app_definitions, $package, $app) = @_;
	
	my $package_ref =  $app_definitions->{$package};
	
	unless (defined $package_ref) {
		die "package $package not found";
	}
	
	my $commands_ref =  $package_ref->{'commands'};
	
	unless (defined $commands_ref) {
		die "commands in $package not found";
	}
	
	
	
	my $app_ref = $commands_ref->{$app};
	
	unless (defined $app_ref) {
		die "app $app not defound";
	}
	
	my $mode_ref = $app_ref->{'default'};
	
	unless (defined $mode_ref) {
		die;
	}
	
	print Dumper($mode_ref);
	
	
	return $mode_ref;
}


sub parse_function_parameters {
	my ($function_def, $app_args_ref) = @_;
	
	
	my @function_arr = split(/\s*,\s*/, $function_def);
	my $key_value_param = {};
	
	my @positional_parameter = ();
	
	
	
	foreach my $param (@function_arr) {
		#print "param: $param\n";
		
		#key - value pair
		if ( my ($value_type, $key, $value) = $param =~ /(\S+)\s+(\S+)\s*\=\s*(\S+)/ ) {
			#print "pair: $value_type, $key, $value \n";
			unless (defined $value) {
				die $param;
			}
			
			if (defined $key_value_param->{$key}) {
				die "key $key already defined";
			}
			
			$key_value_param->{$key}->{'type'} = $value_type;
			$key_value_param->{$key}->{'value'} = $value;
			
			
		} elsif ( ($value_type, $value) = $param =~ /(\S+)\s+(\S+)/ ) {
			#positional or *array or **variable key/value
			
			push(@positional_parameter, [$value_type, $value]);
			#print "positional: $value_type, $value\n";
			
		} elsif ($param =~ /^\*\*/) {
			# indicates **variable key/value
		} elsif ($param =~ /^\*/) {
			
			# put the following pos args into an array until pairs show up
			
		} else {
			die "param $param cannot be parsed";
		}
		
	}
	
	if (@{$app_args_ref} < @positional_parameter ) {
		die "not enough positional arguments";
	}
	
	for (my $i = 0; $i < @positional_parameter; ++$i) {
		my $param = $app_args_ref->[$i];
		
		#my ($param_type, $param_value) = $param =~ /\*(\S+)\s+(\S+)\s*/;
		
		
		my ($value_type, $key) =  @{$positional_parameter[$i]};
		
		
		#unless (lc($param_type) eq lc($value_type)) {
		#	die "types do not match : $param_type vs $value_type";
		#}
		
		
		if (defined $key_value_param->{$key}) {
			die "key $key already defined";
		}
		
		
		$key_value_param->{$key}->{'type'} = $value_type;
		$key_value_param->{$key}->{'value'} = $param;
		
		
		#@app_args
	}
	
	#print Dumper($key_value_param);
	return $key_value_param;
}

sub app {
	my $app_path = shift(@_);
	
	my @app_args = @_;
	
	
	my ($package, $app) = split(/\./, $app_path);
	
	#print "$package -  $app \n";
	unless (defined $app) {
		die "error: $app_path";
		
	}
	
	# get app defintion
	my $mode_ref = get_app_mode($app_definitions, $package, $app);
	
	
	
	
	
	my $function_def = $mode_ref->{'function'};
	
	#print "function_def: $function_def\n";
	
	# use function defintion to get parameters
	my $key_value_param = parse_function_parameters($function_def, \@app_args);
	
	
	
	#replace variable names
	my $command_def = $mode_ref->{'cmd'};
	
	#print "cmd: $command_def\n";
	foreach my $key (keys(%$key_value_param)) {
		my $value = $key_value_param->{$key}->{'value'} || die;
		my $type = $key_value_param->{$key}->{'type'}|| die;
		$command_def =~ s/\[$key\]/$value/g;
		
		if ($type eq 'file') {
			print "input: $key $value\n";
			
		}
		
	}
	
	
	
	#print "cmd: $command_def\n";
	return $command_def;
	
}

