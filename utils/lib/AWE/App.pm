package AWE::App;

use strict;
use warnings;


use parent qw(AWE::Task);
use AWE::TaskOutput;
use AWE::TaskInput;

use Data::Dumper;


use Exporter 'import'; # gives you Exporter's import() method directly
our @EXPORT_OK = qw(app_definitions); # symbols to export on request

our $app_definitions;





1;

sub new {
    my ($class, @args) = @_;
	
    # Call the constructor of the parent class
    my $self = $class->SUPER::new(@args);
	
	
	$self->{'named_inputs_hash'}={};
	
	
	$self->{'output_array'}=[];
	$self->{'output_map'}={};
	
	
	
	if (@args > 0) {
		$self->app(@args);
	}
	
	return $self;
}

# output returns TaskInput objects !!!
sub output {
	my ($self, $arg) = @_;
	
	my $task_out;
	if (wantarray()) {
		$task_out = $self->{'output_array'}->[$arg];
	} else {
		$task_out = $self->{'output_map'}->{$arg}
	}
	
	unless (defined $task_out) {
		die;
	}
	
	my $task_in = new AWE::TaskInput('reference' => $task_out);
	return $task_in;
}

sub addNamedInput {
	my ($self, $input_name, $task_input_obj) = @_;
	
	unless (defined $self->{'named_inputs_hash'}->{$input_name} ) {
		die "error: inputname $input_name unknown to app";
	}
	
	if (ref($self->{'named_inputs_hash'}->{$input_name}) eq 'HASH' ) {
		die "error: input $input_name already assigned"; # TODO may overwrite
	}
	
	$self->{'named_inputs_hash'} = $task_input_obj;
	
	$self->AWE::Task::addInput($task_input_obj);
	
}




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
	my ($self, $function_def, $app_args_ref) = @_;
	
	
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
			
			if ($value_type eq 'file') {
				if (defined $self->{'named_inputs_hash'}->{$value}) {
					die "error: input $value already defined";
				}
				$self->{'named_inputs_hash'}->{$value} = 1;
			}
			
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
		print "expected parameters: ".Dumper(@positional_parameter)."\n";
		print "actual parameters: ".Dumper( @{$app_args_ref})."\n";
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
	my $self = shift(@_);
	#my $app_definitions = shift(@_);
	my $app_path = shift(@_);
	
	my @app_args = @_;
	
	
	my ($package, $app) = split(/\./, $app_path);
	
	#print "$package -  $app \n";
	unless (defined $app) {
		die "error: $app_path";
		
	}
	
	# get app defintion
	my $mode_ref = get_app_mode($app_definitions, $package, $app);
	
	if (defined $app_definitions->{$package}->{'dockerimage'} ) {
		$self->{'cmd'}->{'dockerimage'} = $app_definitions->{$package}->{'dockerimage'};
	}
	
	
	
	my $function_def = $mode_ref->{'function'};
	
	#print "function_def: $function_def\n";
	
	# use function defintion to get parameters
	my $key_value_param = $self->parse_function_parameters($function_def, \@app_args);
	
	#replace variable names
	my $command_def = $mode_ref->{'cmd'};
	
	#print "cmd: $command_def\n";
	# do this two times !
	foreach my $key (keys(%$key_value_param)) {
		my $value = $key_value_param->{$key}->{'value'} || die;
		
		my $type = $key_value_param->{$key}->{'type'}|| die;
		if (ref($value) eq 'AWE::TaskInput' ) {
			if ($type ne 'file') {
				die;
			}
			
			$self->addNamedInput($key, $value); # add input passed as arugment
			
			$value = $value->filename; # use filename in replacements
			
			unless (defined $value) {
				die;
			}
			if ($value eq '') {
				die;
			}
		}
		
		
		$command_def =~ s/\$\{$key\}/$value/g;
		
		if ($type eq 'file') {
			print "input: $key $value\n";
			
		}
		
	}
	
		
	# evaluate variables section
	# example: "variable" : { "OUTPUT-FILE" : "[suffix_remove:INPUT].normalized.fna"},
	my $variables = $mode_ref->{'variable'};
	if (defined $variables) {
		foreach my $variable (keys(%$variables)) {
			my $expression = $variables->{$variable};
			
			my ($prefix, $expr_cmd, $expr_arg, $suffix) = $expression =~ /^(.*)\$\{(.*)\:(.*)\}(.*)$/;
			
			print "prefix: $prefix\n";
			print "expr_cmd $expr_cmd\n";
			print "expr_arg: $expr_arg\n";
			print "suffix: $suffix\n";
			
			if (defined $expr_cmd) {
				if ($expr_cmd eq 'suffix_remove') {
					
					unless ($key_value_param->{$expr_arg}->{'type'} eq 'file') {
						die;
					}
					
					my $file_obj = $key_value_param->{$expr_arg}->{'value'};
					unless (defined $file_obj) {
						die "value for \"$expr_arg\" not found";
					}
					
					my $file=undef;
					if ( ref($file_obj) eq 'AWE::TaskInput' ) {
						# is TaskInput
						
						print "AWE::TaskInput: ".Dumper($file_obj)."\n";
						
						
						$file = $file_obj->filename;
						
						
						print "file (from TaskInput): $file\n";
					} else {
						#print "file (scalar, ".ref($file_obj).")\n";
						$file = $file_obj;
						print "file (scalar): $file\n";
					}
					
					unless (defined $file) {
						die;
					}
					
					# remove suffix
					if ($file eq '') {
						die "filename empty";
					}
					if ($file =~ /\./) {
						$file =~ s/^(.*)\.[^\.]$/$1/;
					}
					print "file: $file\n";
					
					unless (defined $file) {
						die;
					}
					$key_value_param->{$variable}->{'value'} = $prefix.$file.$suffix;
					$key_value_param->{$variable}->{'type'} = 'file';
					
				} else {
					die "expr_cmd \"$expr_cmd\" unknown";
				}
				
			}
			
			
		}
		
	}
	
	
	# create output objects for this task based on app definition
	my $output_array = $mode_ref->{'output_array'};
	
	#print "cmd: $command_def\n";
	# do this two times !
	foreach my $key (keys(%$key_value_param)) {
		my $value = $key_value_param->{$key}->{'value'} || die;
		
		if (ref($value) eq 'AWE::TaskInput' ) {
			$value = $value->filename;
		}
		
		my $type = $key_value_param->{$key}->{'type'}|| die "type of key $key is undefined";
		$command_def =~ s/\$\{$key\}/$value/g;
		
		if ($type eq 'file') {
			print "input: $key $value\n";
			
		}
		
		foreach my $output_file (@{$output_array}) {
			$output_file =~ s/\$\{$key\}/$value/g;
		}
		
	}

	
	my $shockurl = "something";
	
	
	foreach my $output_file (@{$output_array}) {
		
	
		
		my $task_out = new AWE::TaskOutput($output_file, $shockurl);
		$self->addOutput($task_out);
		
		push( @{$self->{'output_array'}}, $task_out);
		
	}
	my $output_map = $mode_ref->{'output_map'};
	
	foreach my $output_file_name (keys(%$output_map)) {
		
		my $output_file = $output_map->{$output_file_name};
		
		my $task_out = new AWE::TaskOutput($output_file, $shockurl);
		$self->addOutput($task_out);
		
		$self->{'output_map'}->{$output_file_name} = $task_out;
		#push( @{$self->{'output_array'}}, $task_out);
		
	}
	
	
	
	
	#make it possible to access positional and named output files
	
	
	#print "cmd: $command_def\n";
	#return $command_def;
	$self->command($command_def);
}

