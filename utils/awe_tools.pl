#!/usr/bin/env perl

use strict;
use warnings;



#https://github.com/MG-RAST/Shock/blob/master/libs/shock.pm
use SHOCK::Client;
use AWE::Client;
use AWE::Job;

use JSON;

use Data::Dumper;
use File::Basename;

eval "use Text::ASCIITable; 1"
or die "module required: cpan install Text::ASCIITable";

use USAGEPOD qw(parse_options);


my $aweserverurl =  $ENV{'AWE_SERVER_URL'};
my $shockurl =  $ENV{'SHOCK_SERVER_URL'};
my $clientgroup = $ENV{'AWE_CLIENT_GROUP'};


my $shocktoken=$ENV{'GLOBUSONLINE'} || $ENV{'KB_AUTH_TOKEN'};

#purpose of wrapper: replace env variables, capture stdout and stderr and archive output directory
my @awe_job_states = ('in-progress', 'completed', 'queued', 'pending', 'deleted' , 'suspend' );









sub wait_for_job {
	
	my ($awe, $job_id) = @_;
	
	my $jobstatus_hash;
	
	# TODO add time out ??
	while (1) {
		sleep(3);
		
		eval {
			$jobstatus_hash = $awe->getJobStatus($job_id);
		};
		if ($@) {
			print "error: getJobStatus $job_id\n";
			exit(1);
		}
		#print $json->pretty->encode( $jobstatus_hash )."\n";
		my $state = $jobstatus_hash->{data}->{state};
		print "state: $state\n";
		if ($state ne 'in-progress') {
			last;
		}
	}
	return $jobstatus_hash;
}










sub getCompletedJobs {
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	
	my $jobs = $awe->getJobQueue('info.clientgroups' => $clientgroup);
	
	#print Dumper($jobs);
	unless (defined $jobs) {
		die;
	}
	unless (defined $jobs->{data}) {
		die;
	}
	
	
	my @completed_jobs =() ;
	
	my $states={};
	$states->{$_}=0 for @awe_job_states;
	foreach my $job (@{$jobs->{data}}){
		$states->{lc($job->{state})}++;
		
		if (lc($job->{state}) eq 'completed') {
			push(@completed_jobs, $job);
		}
		#print "state: ".$job->{state}." user: ".$job->{info}->{user}."\n";
		
		#delete jobs
		if (0) {
			my $dd = $awe->deleteJob($job->{id});
			print Dumper($dd);
		}
		
	}
	
	print "\n** job states **\n";
	print "$_: ".($states->{$_}||'0')."\n" for @awe_job_states;
	if (keys(%$states) > 6) { # in case Wei introduces new states.. ;-)
		die;
	}
	return @completed_jobs;
}




sub getAWE_results_and_cleanup {
	
	my @completed_jobs = getCompletedJobs();
	
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	#print "connect to SHOCK\n";
	my $shock = new SHOCK::Client($shockurl, $shocktoken); # shock production
	unless (defined $shock) {
		die;
	}

	
	my $job_deletion_ok =  AWE::Job::get_jobs_and_cleanup('awe' => $awe, 'shock' => $shock, 'jobs' => \@completed_jobs, 'clientgroup' => $clientgroup);
	
	
	if ($job_deletion_ok == 1) {
		return 1;
	}
	return undef;
}


sub getJobStates {
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	$awe->checkClientGroup($clientgroup);
	
	my $jobs = $awe->getJobQueue('info.clientgroups' => $clientgroup, limit => 1000);
	
	#print Dumper($jobs);
	unless (defined $jobs) {
		die;
	}
	unless (defined $jobs->{data}) {
		die;
	}
	
	my @jobs = ();
	
	my $states={};
	$states->{$_}=[] for @awe_job_states;
	foreach my $job (@{$jobs->{data}}){
		
		my $job_id = $job->{id};
		
		#print Dumper($job)."\n";
		
		my $state = lc($job->{state});
		
		push(@{ $states->{$state} } , $job_id);
		
		
		
	}
	return $states;
}


sub showAWEstatus {
	
	
	
	my $states = getJobStates() || die "states undefined";
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	
	my $jobs = $awe->getJobQueue('info.clientgroups' => $clientgroup, 'limit' => 1000);
	
	#print Dumper($jobs);
	unless (defined $jobs) {
		die;
	}
	unless (defined $jobs->{data}) {
		die;
	}
	
	my @jobs = @{$jobs->{data}};
	
	my @sorted_jobs = sort { $a->{"info"}->{"submittime"} cmp $b->{"info"}->{"submittime"} } @jobs;
	foreach my $job (@sorted_jobs){
		
		my $job_id = $job->{id};
		if ($job->{state} ne "deleted") {
			print $job_id."\t".$job->{"info"}->{"submittime"}."\t".lc($job->{state})."\t".$job->{"info"}->{"name"}."\n";
		}
	}
	
	
	
	print "\n\n** job states **\n";
	#print "$_: ".($states->{$_}||'0')."\n" for @awe_job_states;
	
	foreach my $state (@awe_job_states) {
		if (@{$states->{$state}} > 0) {
			print $state.': '.join(',', @{$states->{$state}})."\n";
		}
	}
	
	
	print "\n\n** job states summary **\n";
	foreach my $state (@awe_job_states) {
		
		print $state.': '.@{$states->{$state}}."\n";
		
	}
	
	if (keys(%$states) > 7) { # in case Wei introduces new states.. ;-)
		print 'keys: '.join(',',keys(%$states));
		die;
	}
	
}


sub drawWorkflow {
    my ($job_obj, $outname, $nolabel) = @_;

	my $edges = {};
	my $id_to_name = {};
	my $task_state = {};
	my $tasks = {};
	    
	foreach my $task (@{$job_obj->{'tasks'}}) {
		my ($task_id) = $task->{'taskid'} =~ /(\d+)$/;
		my $task_name = $task->{'cmd'}->{'name'} || die "no name found";
		$tasks->{$task_id}->{'name'} = $task_name;
		$tasks->{$task_id}->{'state'} = $task->{'state'} || 'unknown';
		print "found $task_id $task_name\n";
		
		my $inputs = $task->{'inputs'};
		foreach my $input ( keys(%$inputs) ) {
			my $origin = $inputs->{$input}->{'origin'};
			if (length($origin) > 10) {
				die "origin length > 10";
			}
			if (defined $origin && $origin ne '') {
				$edges->{$origin}->{$task_id}->{$input}=1;
			}
		}
		foreach my $origin ( @{$task->{'dependsOn'}} ) {
			if (length($origin) > 10) {
				print  "origin length > 10, $origin";
				my ($origintask) =$origin =~ /\_(\d+)$/;
				$origin = $origintask;
			}
		    unless (exists $edges->{$origin}->{$task_id}) {
		        $edges->{$origin}->{$task_id}->{'virtual'}=1;
	        }
		}
	}
    
	print "edges: ".Dumper($edges)."\n";
	my @graph = ('digraph G {');
	
	foreach my $t (keys(%$tasks)) {
		if (length($t) > 10) {
			die "task length > 10";
		}
		my $state =$tasks->{$t}->{'state'};
		my $name =$tasks->{$t}->{'name'};
		my $color = 'white';
		if ($state eq 'completed') {
			$color = 'green';
		} elsif ($state eq 'suspended') {
			$color = 'red';
		} elsif ($state eq 'in-progress') {
			$color = 'darkorchid1'
		} elsif ($state eq 'queued') {
			$color = 'cadetblue1'
		} elsif ($state eq 'pending') {
			$color = 'cadetblue1'
		}
		$name .= " ($t)";
		push (@graph, "\t".$t.' [label="'.$name.'", style=filled, fillcolor='.$color.']');
	}
	
	foreach my $from (keys(%{$edges})) {
		if (length($from) > 10) {
			die "from length > 10";
		}
		my $tos_ref  = $edges->{$from};
		my @tos = keys(%{$tos_ref});
		
		foreach my $to (@tos) {
			print "$from -> $to\n";
			foreach my $label (keys(%{$tos_ref->{$to}})) {
				my $e = "";
				if (($label eq 'virtual') || $nolabel) {
				    $e = "\t$from -> $to;";
				} else {
				    $e = "\t$from -> $to [taillabel = \"$label\"];";
				}
				print $e."\n";
				push (@graph, $e);
			}
		}	
	}
	
	push (@graph, '}');
	open FILE, '>'.$outname.'.dot' or die $!;
	print FILE join("\n", @graph)."\n";
	close FILE;
	
	my $cmd = 'dot -Tpng '.$outname.'.dot -o '.$outname.'.png';
	print "cmd: $cmd\n";
	system($cmd);
}



########################################################################################
########################################################################################
########################################################################################
########################################################################################
# START





my ($h, $help_text) = &parse_options (
	'name' => 'awe_cli',
	'version' => '1',
	'synopsis' => 'awe_cli --status',
	'examples' => 'ls',
	'authors' => 'Wolfgang Gerlach',
	'options' => [
		'',
		'Actions:',
		[ 'status'						, "show job states on AWE server"],
		[ 'cmd=s'						, "command to execute"],
        [ 'show_clients'                , "show clients in current clientgroup"],
		[ 'resume_clients=s'            , " "],
		[ 'show_jobs=s'					, " "],
		[ 'check_jobs=s'				, " "],
		[ 'download_jobs=s'				, "download specified jobs if state==completed"],
		[ 'wait_and_download_jobs=s'	, "wait for completion and download specified jobs"],
		[ 'resume_jobs=s'				, "resume jobs"],
		[ 'suspend_jobs=s'				, "suspend jobs"],
		[ 'resubmit_jobs=s'				, "resubmit jobs"],
		[ 'delete_jobs=s'				, "deletes jobs and temporary shock nodes, unless keep_nodes used"],
		[ 'shock_clean'					, "delete all temporary nodes from SHOCK server"],
		[ 'shock_query=s'				, "query SHOCK node"],
		[ 'shock_view=s'                , "view SHOCK node"],
		[ 'draw_from_ids=s'				, "uses dot (GraphViz) to draw workflow graph from one or more job ids"],
		[ 'draw_from_file=s'			, "uses dot (GraphViz) to draw workflow graph from a file"],
		'',
		'Drawing Options',
		[ 'nolabel'						, "do not show labels in graph"],
		'',
		'Options:',
		[ 'out_dir=s'						, "specify download directory (default uses job name)"],
		[ 'keep_nodes'					, "use with --delete_jobs"],
		[ 'wait_completed'				, "wait until any job is in state completed"],
		[ 'output_files=s'				, "specify extra output files for --cmd"],
		[ 'clientgroup=s',				, "clientgroup"],
		[ 'aweserver=s',				, "AWE serverl url"],
		[ 'help|h'						, "", { hidden => 1  }]
	]
);



if ($h->{'help'} || keys(%$h)==0) {
	print $help_text;
	exit(0);
}

if (defined($h->{'aweserver'})) {
	$aweserverurl = $h->{'aweserver'};
}

if (defined($h->{'clientgroup'})) {
	$clientgroup = $h->{'clientgroup'};
}

print "Configuration:\n";
print "aweserverurl: ".($aweserverurl || 'undef') ."\n";
print "shockurl: ". ($shockurl || 'undef') ."\n";
print "clientgroup: ". ($clientgroup || 'undef') ."\n\n";


if (defined($h->{"status"})) {
	showAWEstatus();
	exit(0);
} elsif (defined($h->{"show_clients"})) {
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	#$awe->checkClientGroup($clientgroup);
	
	my $clients = $awe->request('GET', 'client');
	
	#print Dumper($clients);
	#print "clientgroup: $clientgroup\n";
	
	my $t = Text::ASCIITable->new({ headingText => "clientgroup: $clientgroup" });
	
	$t->setCols('ID', 'name', 'IP', 'status', 'current work');
	#$t->alignCol('networks','left');
	
	
	
	foreach my $client (@{$clients->{'data'}}) {
		
		if (lc($client->{'group'}) eq lc($clientgroup)) {
			#print Dumper($client);
			#print $client->{'name'}."\t";
			#print ($client->{'Status'} || 'NA');
			#print "\t";
			
			my @current = keys(%{$client->{'current_work'}});
			my $cur_text;
			if (@current == 0 ) {
				$cur_text = "idle";
			} else {
				$cur_text = join(',', @current);
			}
			
			$t->addRow( $client->{'id'}, $client->{'name'}, $client->{'host'}, ($client->{'Status'} || 'NA'), $cur_text);
		}
		
		
	}
	
	print $t;
	
	exit(0);
} elsif (defined($h->{"resume_clients"})) {
	my @clients = split(',', $h->{"resume_clients"});
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	foreach my $client (@clients) {
		$awe->resumeClient($client);
	}

	
	
	exit(0);
} elsif (defined($h->{"wait_completed"})) {
	
	my $completed = 0;
	while (1) {
		my $states = getJobStates();
		$completed = @{$states->{'completed'}};
		if ($completed > 0) {
			last;
		}
		sleep(5);
	}
	
	print "$completed job(s) are in state \"complete\"\n";
	exit(0);
} elsif (defined($h->{"wait_and_download_jobs"})) {
	
	my @jobs = split(',', $h->{"wait_and_download_jobs"});
	
	############################################
	# connect to AWE server and check the clients
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	
	$awe->checkClientGroup($clientgroup) == 0 or exit(1);
	
	############################################
	#connect to SHOCK server
	
	print "connect to SHOCK\n";
	my $shock = new SHOCK::Client($shockurl, $shocktoken); # shock production
	unless (defined $shock) {
		die;
	}
	
	AWE::Job::wait_and_download_job_results ('awe' => $awe, 'shock' => $shock, 'jobs' => \@jobs, 'clientgroup' => $clientgroup);
	
	exit(0);

} elsif (defined($h->{"delete_jobs"})) {
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	my @jobs = split(',', $h->{"delete_jobs"});
	
	if (defined($h->{"keep_nodes"})) { # delete jobs without deleteing shock nodes
		foreach my $job (@jobs) {
			my $dd = $awe->deleteJob($job);
			print Dumper($dd);
		}
	} else {
		
		print "connect to SHOCK\n";
		my $shock = new SHOCK::Client($shockurl, $shocktoken); # shock production
		unless (defined $shock) {
			die;
		}
		
		
		AWE::Job::delete_jobs ('awe' => $awe, 'shock' => $shock, 'jobs' => \@jobs, 'clientgroup' => $clientgroup);
		
	}
	exit(0);
} elsif (defined($h->{"resume_jobs"})) {
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	my @jobs = split(',', $h->{"resume_jobs"});
	foreach my $job (@jobs) {
		$awe->resumeJob($job);
	}
	
	
	
	exit(0);
} elsif (defined($h->{"suspend_jobs"})) {
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	my @jobs = split(',', $h->{"suspend_jobs"});
	foreach my $job (@jobs) {
		$awe->suspendJob($job);
	}
	
	
	
	exit(0);
} elsif (defined($h->{"resubmit_jobs"})) {
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	my @jobs = split(',', $h->{"resubmit_jobs"});
	foreach my $job (@jobs) {
		$awe->resubmitJob($job);
	}
	
	
	
	exit(0);
} elsif (defined($h->{"shock_query"})) {
	
	
	my @queries = split(',', $h->{"shock_query"});
	
	my $shock = new SHOCK::Client($shockurl, $shocktoken);
	unless (defined $shock) {
		die;
	}
	
	my $response =  $shock->query(@queries);
	print Dumper($response);
	exit(0);

} elsif (defined($h->{"shock_view"})) {
	
	
	my @nodes = split(',', $h->{"shock_view"});
	
	my $shock = new SHOCK::Client($shockurl, $shocktoken);
	unless (defined $shock) {
		die;
	}
	
	foreach my $node (@nodes) {
		my $response =  $shock->get('node/'.$node);
		print Dumper($response);
	}
	
	
	
	exit(0);
	
} elsif (defined($h->{"shock_clean"})) {
	
	my $shock = new SHOCK::Client($shockurl, $shocktoken);
	unless (defined $shock) {
		die;
	}
		
	my $response =  $shock->query('temporary' => 1);
	
	#my $response =  $shock->query('statistics.length_max' => 1175);
	print Dumper($response);
	#exit(0);
	
	my @list =();
	
	unless (defined $response->{'data'}) {
		die;
	}
	
	foreach my $node (@{$response->{'data'}}) {
		#print $node->{'id'}."\n";
		push(@list, $node->{'id'});
	}
	
	print "found ".@list. " nodes that can be deleted\n";
	
	foreach my $node (@list) {
		my $ret = $shock->delete_node($node);
		defined($ret) or die;
		print Dumper($ret);
	}
	
	
	exit(0);

} elsif (defined($h->{"download_jobs"})) {
	
	my @jobs = split(/,/,$h->{"download_jobs"});
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	my $shock = new SHOCK::Client($shockurl, $shocktoken); # shock production
	unless (defined $shock) {
		die;
	}
	
	
	
	my $only_last_task = 0;
	
	AWE::Job::download_jobs('awe' => $awe, 'shock' => $shock, 'jobs' => \@jobs, 'use_download_dir' => $h->{'out_dir'}, 'only_last_task' => $only_last_task);
	
	print "done.\n";
	exit(0);
	
} elsif (defined($h->{"check_jobs"})) {
		
	my @jobs = split(/,/,$h->{"check_jobs"});
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	my $ret = AWE::Job::check_jobs('awe' => $awe,  'jobs' => \@jobs, 'clientgroup' => $clientgroup);
	
	if ($ret ==0) {
		print "all jobs completed :-) \n";
		exit(0);
	}

	print "jobs not completed :-( \n";
	exit(1);
} elsif (defined($h->{"show_jobs"})) {
	
	my @jobs = split(/,/,$h->{"show_jobs"});
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	foreach my $job (@jobs) {
		my $job_obj = $awe->getJobStatus($job);
		print Dumper($job_obj);
	}
	
	exit(0);
} elsif (defined($h->{"draw_from_ids"})) {

	my @jobs = split(/,/,$h->{"draw_from_ids"});
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	foreach my $job (@jobs) {
		my $job_obj = $awe->getJobStatus($job);
		drawWorkflow($job_obj->{'data'}, $job, $h->{"nolabel"});
	}
	exit(0);

} elsif (defined($h->{"draw_from_file"})) {
    
    my $json = JSON->new;
    my $job_obj = {};
    
    unless (-s $h->{"draw_from_file"}) {
        die;
    }
    open(IN, "<".$h->{"draw_from_file"}) or die "Couldn't open file: $!";
    $job_obj = $json->decode(join("", <IN>)); 
    close(IN);
    
	drawWorkflow($job_obj, $h->{"draw_from_file"}, 1);
	exit(0);

} elsif (defined($h->{"get_all"})) {
	#getAWE_results_and_cleanup();
	
	
	
	print "done.\n";
	exit(0);
} elsif (defined($h->{"cmd"})) {
	
	my $cmd = $h->{"cmd"};
	my $output_files = $h->{"output_files"};
	
	my @moreopts=();
	if (defined $output_files) {
		@moreopts = ("output_files", $output_files);
	}
	
	
	#example:
	#./awe.pl --output_files=ucr/otu_table.biom --cmd="pick_closed_reference_otus.py -i @4506694.3.fas -o ucrC97 -p @otu_picking_params_97.txt -r /home/ubuntu/data/gg_13_5_otus/rep_set/97_otus.fasta"
	
	
	############################################
	# connect to AWE server and check the clients
	
	my $awe = new AWE::Client($aweserverurl, $shocktoken);
	unless (defined $awe) {
		die;
	}
	
	$awe->checkClientGroup($clientgroup) == 0 or exit(1);
	
	
	############################################
	#connect to SHOCK server
	
	print "connect to SHOCK\n";
	my $shock = new SHOCK::Client($shockurl, $shocktoken); # shock production
	unless (defined $shock) {
		die;
	}

	
	## compose AWE job ##
	AWE::Job::generateAndSubmitSimpleAWEJob('cmd' => $cmd, 'awe' => $awe, 'shock' => $shock, 'clientgroup' => $clientgroup, @moreopts);
	
	
	
	exit(0);

}




