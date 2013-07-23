#!/usr/bin/env perl 
#Pipeline job submitter for AWE
#Command name: awe_submit.pl
#Usage: see the bottom of the file

use strict;
use warnings;
no warnings('once');

use Getopt::Long;
use LWP::UserAgent;
use JSON;
use Data::Dumper;
use File::Copy;
use File::Basename;
use POSIX qw(strftime);
umask 000;

# options
my %info = ();
$info{user} = "default";
$info{project} = "default";
$info{jobname} = "dafault";
$info{queue} = "default";

my $infiles_str = "";
my $infile_base = "";
my $awe_url = "";
my $shock_url = "";
my $node_id = "";
my $awf = "";
my $input_script = "";
my $total_work=1;
my $file_type="";
my $var_str="";
my $help = 0;

my $infile_name = "";

my %vars = ();
my %infmap = ();
    
my @inf_names = ();
my @shock_ids = ();

my $options = GetOptions ("uploads=s"   => \$infiles_str,
                          "awe=s"    => \$awe_url,
 			  "shock=s"  => \$shock_url,
                          "nodes=s"  => \$node_id,
			  "type=s" => \$file_type,
			  "awf=s" => \$awf,
                          "script=s"   => \$input_script,
                          "user=s"   => \$info{user},
                          "project=s" => \$info{project},
                          "name=s" => \$info{jobname},
                          "queue=s" => \$info{queue},
                          "totalwork=i" => \$total_work,
			  "vars=s" => \$var_str,
                          "h"  => \$help,
			 );

my $jobscript = "";

if ($help) {
    print_usage();
    exit 0;
}

if (length($awe_url)==0) {
    $awe_url = $ENV{'AWE_HOST'};
    if (length($awe_url)==0) {
        print "ERROR: AWE server URL was not specified.\n";
        print_usage();
        exit 1;
    }
}

if (length($infiles_str)>0 && length($input_script)>0) {
    print "ERROR: input file and json script are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($infiles_str)>0 && length($input_script)>0) {
    print "ERROR: local input file and job script are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($infiles_str)>0 && length($node_id)>0) {
    print "ERROR: shock node and local input file are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($input_script)>0 && length($node_id)>0) {
    print "ERROR: shock node and job script are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($infiles_str)==0 && length($input_script)==0 && length($node_id)==0) {
    print "ERROR: please specify either of the following: shock node id (-node), local path of the input file (-upload), or the exiting job json script (-script)\n";
    print_usage();
    exit 1;
}

if (length($node_id)>0 && length($file_type)==0) {
    print "ERROR: please specify file type (fasta fastq fna) when using shock node as input\n";
    print_usage();
    exit 1;
}

if (length($node_id)>0 || (length($infiles_str)>0)) { #use case 1 or 2
    if (length($shock_url)==0 ) {
        $shock_url = $ENV{'SHOCK_HOST'};
        if (length($shock_url)==0) {
            print "ERROR: SHOCK server URL was not specified.\n";
            print_usage();
            exit 1;
        }
    }
    if (length($awf)==0){
        print "ERROR: a pipeline template was not specified.\n";
        print_usage();
        exit 1;  
    }elsif (! -e $awf){
        print "ERROR: The pipeline template file [$awf] does not exist.\n";
        print_usage();
        exit 1;  
    }
        
    #generate job script based on template (instantiate a job script with availalbe information filled into the template)
    my $workflow_name = `basename $awf`;
    chomp($workflow_name);
    print "workflow_name=".$workflow_name."\n";
    if ($info{jobname} eq "default") {
        $info{jobname} = $workflow_name."-job";
    }
    print "job_name=".$info{jobname}."\n";
    $jobscript = $workflow_name.".json";
    system("cp $awf $jobscript");
    my $json = new JSON();
    my $awfjson = read_from_file($jobscript);
    
    foreach my $key (keys %info) {
	$awfjson->{job_info}->{$key} = $info{$key};
    }
   
    my $infile_count = scalar(keys  %{$awfjson->{raw_inputs}});
    
    
    if (length($infiles_str)>0) {  #input file is local, upload it (use case 2)
	my @splits = split ",", $infiles_str;
	my $ct =  scalar @splits;
	if ($ct != $infile_count) {
	    print "ERROR: number of inputs in job submission doesn't match number of inputs defined in workflow:".$ct." vs ".$infile_count."\n";
	    exit(1);
	}

	for my $inf (split ",", $infiles_str) {
	    if (! -e $inf){ 
                print "ERROR: The input genome file [$inf] does not exist.\n";
                print_usage();
                exit 1;  
            }
            #upload input to shock
            print "uploading input file ".$inf." to Shock...\n";
            my $ua = LWP::UserAgent->new();
            my $post = $ua->post("http://".$shock_url."/node",
                     Content_Type => 'form-data',
                     Content      => [ upload => [$inf] ]
                    );

            my $json = new JSON();
            my $res = $json->decode( $post->content );
            my $shock_id = $res->{data}->{id};
	    my $inf_name = $res->{data}->{file}->{name};
	    
	    push(@inf_names, $inf_name);
	    push(@shock_ids, $shock_id);
	    
            print "\nuploading input file to Shock...Done! shock node id=".$shock_id."\n";
	}	
    } else { #input file is already on shock and the node id is specified by -node (use case 1)
	my @splits = split ",", $node_id;
	my $ct =  scalar @splits;
	if ($ct != $infile_count) {
	    print "ERROR: number of inputs in job submission doesn't match number of inputs defined in workflow:".$ct." vs ".$infile_count."\n";
	    exit(1);
	}
	for my $id (split ",", $node_id) {
	    push(@shock_ids, $id);
	}
    }
    
    
    %{$awfjson->{raw_inputs}} = ();
    
    for (my $i = 1; $i <= $infile_count; $i++)  {
        print "input #".$i."\n";
	my $keyname = "#i_".$i;
	my $infname = $inf_names[$i-1];
	$infmap{$keyname} = $infname;
	my $input_location = 'http://'.$shock_url.'/node/'.$shock_ids[$i-1]."?download";
        $awfjson->{raw_inputs}->{$infname} = $input_location;
    }

    if (length($var_str)>0) {
        #parse variable string
	for my $ov (split ":", $var_str) {
	    my $option = "";
	    my $value = "";
            if ($ov =~ /=/) {
                ($option, $value) = split "=", $ov;
            } else {
		$option = $ov;
		$value = "-".$ov;
            }
	    $vars{$option} = $value;
        }
    }
    
    foreach my $key (keys %vars) {
        $awfjson->{variables}->{$key} = $vars{$key};
    }
    
    $awfjson->{data_server} = 'http://'.$shock_url;
    
    open my $fh, ">", $jobscript;
    print $fh encode_json($awfjson);
    close $fh;
    
    foreach my $key (keys %infmap) {
	my $value = $infmap{$key};
	system("perl -p -i -e 's/$key/$value/g;' $jobscript");
	print "input name ".$key." changed to name ".$value."\n";
    }
    
} else { #Use case 3, use the already instantiated job script
    $jobscript = $input_script;
}

#upload job script to awe server
system("stat $jobscript");
print "submitting job script to AWE...jobscript=$jobscript \n";
my $out_awe_sub = `curl -X POST -F awf=\@$jobscript http://$awe_url/job | python -mjson.tool |  grep \\\"id\\\"`;
if($? != 0) {
    print "Error: Failed to submit job script to AWE server, return value $?\n";
    exit $?;
}
chomp($out_awe_sub);
my @values = split('"', $out_awe_sub);
my $job_id = $values[3];


print "\nsubmitting job script to AWE...Done! id=".$job_id."\n";

print "job submission summary:\n";
print "pipeline job awe url: http://".$awe_url."/job/".$job_id."\n";

if (length($infiles_str)>0 || length($node_id)>0) {
    my $refjson = "awf_".$job_id.".json"; 
    system("mv $jobscript $refjson");
    print "job script for reference: $refjson\n";
} 

exit(0);

sub read_from_file {
    my $json;
    {
         local $/; #Enable 'slurp' mode
         open my $fh, "<", $_[0];
         $json = <$fh>;
         close $fh;
    }
    return decode_json($json);
}


sub print_usage{
    print "
Job submitter for AWE (using .awf)
Command name: awf_submit.pl
Options:
     -awe=<AWE server URL (ip:port), required>
     -shock=<Shock URL (ip:port), required>
     -node=<shock node id of the input file, supporting multiple node ids separated by ','>
     -type=<input file type, fna|fasta|fastq|fa, required when -node is set>
     -upload=<input files that is local and to be uploaded, supporting multiple file separated by ','>
     -awf=<path for awf workflow definition, required when -node or -upload is set>
     -script=<path for complete job json file>
     -name=<job name>
     -user=<user name>
     -project=<project name>
     -queue=<queue names (separate by ',')>
     -totalwork=<number of workunits to split for splitable tasks (default 1)>
     
Use case 1: submit a job with a shock url for the input file location and a pipeline template (input file is on shock)
      Required options: -node, -awf, -awe (if AWE_HOST not in ENV), -shock (if SHOCK_HOST not in ENV)
      Optional options: -name, -user, -project, -cgroups, -totalwork
      Operations:
               1. create job script based on job template and available info
               2. submit the job json script to awe

Use case 2: submit a job with a local input file and a pipeline template (input file is local and will be uploaded to shock automatially;
      Required options: -upload, -awf, -awe (if AWE_HOST not in ENV), -shock (if SHOCK_HOST not in ENV)
      Optional options: -name, -user, -project, -cgroups, -totalwork
      Operations:
               1. upload input file to shock
               2. create job script based on job template and available info
               3. submit the job json script to awe
               
Use case 3: submit a job with a complete job json script (job script is already instantiated, suitable for recomputation)
      Required options: -script, -awe
      Optional options: none  (all needed info is in the job script)
      Operations: submit the job json script to awe directly.
      
note:
1. the three use cases are mutual exclusive: at least one and only one of -node, -upload, and -upload can be specified at one time.
2. if AWE_HOST (ip:port) and SHOCK_HOST (ip:port) are configured as environment variables, -awe and -shock are not needed respectively. But
the specified -awe and -shock will over write the preconfigured environment variables.
\n";
}


