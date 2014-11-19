package AWE::Workflow;
use base qw(Exporter);

use strict;
use warnings;

use AWE::Task;
use AWE::App;
use AWE::TaskInput;
use AWE::TaskOutput;


use SHOCK::Client;

use JSON;

use Data::Dumper;



our @ISA    = qw(Exporter); # Use our.
# Exporting the add and subtract routine
our @EXPORT = qw(list_resource shock_resource task_resource string_resource);



sub new {
    my ($class, %h) = @_;
    
	my $info = {
		"pipeline"		=> ($h{'pipeline'} || "#pipeline"),
		"name"			=> ($h{'name'} || "#jobname"),
		"project"		=> ($h{'project'} || "#project"),
		"user"			=> ($h{'user'} || "#user"),
		"clientgroups"	=> ($h{'clientgroups'} || "#clientgroups"),
		"noretry"		=> ($h{'noretry'} || JSON::true)
	};
	
	my $tasks = [];
	
    my $self = {
		info => $info,
		tasks => $tasks,
		shockhost => $h{'shockhost'},
		shocktoken => $h{'shocktoken'}
	};
	
    bless $self, $class;
    return $self;
}

sub shockhost {
	my ($self) = @_;
	return $self->{shockhost};
}


sub shocktoken {
	my ($self) = @_;
	return $self->{shocktoken};
}

sub info {
    my ($self) = @_;
    return $self->{info};
}

sub tasks {
    my ($self) = @_;
    return $self->{tasks};
}

sub taskcount {
	my ($self) = @_;
    return scalar(@{$self->{tasks}});
}

# add task to workflow and return task with taskid assigned (see newTask)
sub addTask {
	my ($self, $task) = @_;
	
	push(@{$self->tasks}, $task);
	
	
	my $taskid = scalar(@{$self->tasks})-1;
	
	$task->taskid($taskid."");
	
	return $task;
}

#creates, adds, and return a new task
sub newTask {
	my ($self, $name, @app_args) = @_;
	my $newtask = new AWE::Task();
	
	unless (defined $name) {
		die "app name not defined";
	}
	$newtask->{'cmd'}->{'name'} = $name;
	
	for (my $i=0; $i < @app_args ; $i++) {
		my $res = $app_args[$i];
		unless (defined $res->{'resource'}) {
			die "resource type not defined";
		}
		
		
		if (lc($res->{'resource'}) eq 'shock') { # TODO this is a feature that should be handled by AWE
			$self->addFilenameToResource($res);
			unless (defined $res->{'resource'}) {
				die;
			}
		} elsif (lc($res->{'resource'}) eq 'list') {
			my $list = $res->{'list'};
			foreach my $elem (@{$list}) {
				if (lc($elem->{'resource'}) eq 'shock') {
					$self->addFilenameToResource($elem);
				}
				
			}
			
		}
		
	}
	
	
	$newtask->{'cmd'}->{'app_args'} = \@app_args;
	
	
	return $self->addTask($newtask);
}

sub addFilenameToResource {
	my ($self, $res) = @_;
	
	my $filename = $res->{'filename'};
	my $host = $res->{'host'};
	my $node = $res->{'node'};
	unless (defined $filename && $filename ne "") {
		
		
		#print "token: ". $self->shocktoken(). "\n";
		
		my $shock = new SHOCK::Client($host, $self->shocktoken(), 0);
		
		my $obj = $shock->get_node($node);
		unless (defined $obj) {
			die "could not read shock node";
		}
		
		my $node_filename = "";
		if (defined $obj) {
			$node_filename = $obj->{data}->{file}->{name};
		}
		
		unless (defined $node_filename) {
			$node_filename = "";
		}
		if ($node_filename eq "") {
			
			print Dumper($obj);
			print $node."\n";
			die;
			
			$node_filename = $node.".dat";
		}
		$filename = $node_filename;
		$res->{'filename'} =$filename;
	}
	return;
}

# add tasks and assign taskids
sub addTasks {
	my ($self) = shift(@_);
	
	my @tasks = @_;
	
	foreach my $task (@tasks) {
		push(@{$self->tasks}, $task);
		my $taskid = scalar(@{$self->tasks})-1;
		$task->taskid($taskid."");
	}
	return;
}




sub getHash {
	my ($self) = @_;
	
	my $wf = {};
	
	$wf->{'info'} = $self->info;
	
	my $tasks_ = [];
	
	foreach my $task (@{$self->tasks}) {
		push(@{$tasks_}, $task->getHash());
	}
	
	$wf->{'tasks'} = $tasks_;
	
	
	$wf->{'shockhost'} = $self->shockhost();
	
	return $wf;
}

#upload data and files in the workflow to SHOCK (marked as temporary nodes!)
sub shock_upload {
	my ($self, $shockurl, $shocktoken) = @_;

	
	require SHOCK::Client;
	
	print "connect to SHOCK\n";
	my $shock = new SHOCK::Client($shockurl, $shocktoken); # shock production
	unless (defined $shock) {
		die;
	}
	
	
	my $uploaded_data={};
	my $uploaded_files={};
	
	my $attr = '{"temporary" : "1"}';
	
	foreach my $task ( @{$self->tasks} ) {
		foreach my $input ( @{$task->inputs} ) {
			my $node_object=undef;
			
			my $data = $input->data;
			my $localfile = $input->localfile;
			
			my $filename = $input->filename;
			if (defined $data) {
				if (defined $uploaded_data->{$data}) {
					$input->node($uploaded_data->{$data});
					$input->host($shockurl);
					next;
				}
				
				print 'upload data '.$filename."\n";
				$node_object= $shock->upload('data' => $$data, 'attr' => $attr);
				
			} elsif (defined  $localfile) {
				if (defined $uploaded_files->{$localfile}) {
					$input->node($uploaded_files->{$localfile});
					$input->host($shockurl);
					next;
				}

				print 'upload local file '.$filename."\n";
				$node_object = $shock->upload('file' => $localfile, 'attr' => $attr);
				
			} else {
				
				$input->reference_update();
				
				
				next;
			}
			
			unless (defined $node_object) {
				die "error: node_object not defined";
			}
			
			my $node = $node_object->{'data'}->{'id'};
			
			unless (defined $node) {
				die "error: node id not found";
			}
				
			$input->node($node);
			$input->host($shockurl);
			
			if (defined $data ) {
				$uploaded_data->{$data}=$node;
			} else {
				$uploaded_files->{$localfile}=$node;
			}
			
			
			
		}
		
		
		
	}
	
	return;
}


sub list_resource {
	my ($list_ref) = @_;

	
	my $res = {	"resource" => "list",
		"list" => $list_ref
	};
	
}

sub shock_resource {
	my ($host, $node, $filename) = @_;
	
	unless (defined $node) {
		#http://shock.metagenomics.anl.gov/node/80cce328-f8ce-4f2c-b189-803ac12f9e44
		my ($h, $n) =$host =~ /^(.*)\/node\/(.*)(\?download)?/;
		
		unless (defined $n) {
			die "could not parse shock url";
		}
		$host = $h;
		$node = $n;
		
	}
	
	
	my $res = {"resource" => "shock",
		"host" => $host,
		"node" => $node};
	
	
	if (defined $filename) {
		$res->{"filename"} = $filename ;
	}
	return $res;
}


# key-value pair or only value
sub string_resource {
	my ($key, $value) = @_;
	
	unless (defined $value) {
		$value = $key;
	}
	
	my $res = {"resource" => "string",
				"value" => $value
	};
	
	if (defined $key) {
		$res->{'key'} = $key;
	}
	
	return $res;
}



sub task_resource {
	my ($task, $pos, $host) = @_;
	
	my $res = {"resource" => "task",
		"task" => $task, #string
		"position" => $pos, # number;
		"host" => $host
	};
	
	
	return $res;
}

1;
