package AWE::Workflow;

use strict;
use warnings;

use JSON;



sub new {
    my ($class, %h) = @_;
    
	my $info = {
		"pipeline"		=> $h{'pipeline'} || "#pipeline",
		"name"			=> $h{'name'} || "#jobname",
		"project"		=> $h{'project'} || "#project",
		"user"			=> $h{'user'} || "#user",
		"clientgroups"	=> $h{'clientgroups'} || "#clientgroups",
		"noretry"		=> $h{'noretry'} || JSON::true
	};
	
	my $tasks = [];
	
    my $self = {
		info => $info,
		tasks => $tasks
	};
	
    bless $self, $class;
    return $self;
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

# add task to workflow and return task
sub addTask {
	my ($self, $task) = @_;
	
	push(@{$self->tasks}, $task);
	
	
	my $taskid = scalar(@{$self->tasks})-1;
	
	$task->taskid($taskid."");
	
	return $task;
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


1;
