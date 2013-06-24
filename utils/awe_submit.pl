#!/usr/bin/env perl 
#Pipeline job submitter for AWE
#Command name: awe_submit.pl
#Options:
#     -awe=<AWE server URL (ip:port), required>
#     -shock=<Shock URL (ip:port), required>
#     -node=<shock node of the input file>
#     -type=<input file type, fna|fasta|fastq|fa, required when -node is set>
#     -upload=<input file that is local and to be uploaded>
#     -pipeline=<path for pipeline job template, required when -node or -upload is set>
#     -script=<path for complete job json file>
#     -name=<job name>
#     -user=<user name>
#     -project=<project name>
#     -cgroups=<exclusive_client_group_list (separate by ',')>
#     -totalwork=<number of workunits to split for splitable tasks (default 1)>
#     
#     
#Use case 1: submit a job with a shock url for the input file location and a pipeline template (input file is on shock)
#      Required options: -node, -pipeline, -awe (if AWE_HOST not in ENV), -shock (if SHOCK_HOST not in ENV)
#      Optional options: -name, -user, -project, -cgroups
#      Operations:
#               1. create job script based on job template and available info
#               2. submit the job json script to awe
#
#Use case 2: submit a job with a local input file and a pipeline template (input file is local and will be uploaded to shock automatially;
#      Required options: -upload, -pipeline, -awe (if AWE_HOST not in ENV), -shock (if SHOCK_HOST not in ENV)
#      Optional options: -name, -user, -project, -cgroups
#      Operations:
#               1. upload input file to shock
#               2. create job script based on job template and available info
#               3. submit the job json script to awe
#               
#Use case 3: submit a job with a complete job json script (job script is already instantiated, suitable for recomputation)
#      Required options: -script, -awe
#      Optional options: none  (all needed info is in the job script)
#      Operations: submit the job json script to awe directly.
#      
#note:
#1. the three use cases are mutual exclusive: at least one and only one of -node, -upload, and -upload can be specified at one time.
#2. if AWE_HOST (ip:port) and SHOCK_HOST (ip:port) are configured as environment variables, -awe and -shock are not needed respectively. But
#the specified -awe and -shock will over write the preconfigured environment variables.

use strict;
use warnings;
no warnings('once');

use Getopt::Long;
use File::Copy;
use File::Basename;
use POSIX qw(strftime);
umask 000;

# options
my $infile = "";
my $infile_base = "";
my $awe_url = "";
my $shock_url = "";
my $node_id = "";
my $pipeline_template = "";
my $input_script = "";
my $user_name = "default";
my $project_name = "default";
my $job_name="";
my $clients="";
my $total_work=1;
my $file_type="";
my $help = 0;

my $options = GetOptions ("upload=s"   => \$infile,
                          "awe=s"    => \$awe_url,
 			  "shock=s"  => \$shock_url,
                          "node=s"  => \$node_id,
			  "pipeline=s" => \$pipeline_template,
                          "script=s"   => \$input_script,
                          "user=s"   => \$user_name,
                          "project=s" => \$project_name,
                          "name=s" => \$job_name,
                          "type=s" => \$file_type,
                          "cgroups=s" => \$clients,
                          "totalwork=i" => \$total_work,
                          "h"  => \$help,
			 );

my $jobscript = "";
my $shock_id = "";

if ($help) {
    print_usage();
    exit 0;
}

print "total_work=".$total_work;

if (length($awe_url)==0) {
    $awe_url = $ENV{'AWE_HOST'};
    if (length($awe_url)==0) {
        print "ERROR: AWE server URL was not specified.\n";
        print_usage();
        exit 1;
    }
}

if (length($infile)>0 && length($input_script)>0) {
    print "ERROR: input file and json script are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($infile)>0 && length($input_script)>0) {
    print "ERROR: local input file and job script are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($infile)>0 && length($node_id)>0) {
    print "ERROR: shock node and local input file are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($input_script)>0 && length($node_id)>0) {
    print "ERROR: shock node and job script are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($infile)==0 && length($input_script)==0 && length($node_id)==0) {
    print "ERROR: please specify either of the following: shock node id or local path of the input file, or the exiting job json script\n";
    print_usage();
    exit 1;
}

if (length($node_id)>0 && length($file_type)==0) {
    print "ERROR: please specify file type (fasta fastq fna) when using shock node as input\n";
    print_usage();
    exit 1;
}

if (length($infile)>0) {
    $infile_base = basename($infile)
}


if (length($node_id)>0 || (length($infile)>0)) { #use case 1 or 2
    if (length($shock_url)==0 ) {
        $shock_url = $ENV{'SHOCK_HOST'};
        if (length($shock_url)==0) {
            print "ERROR: SHOCK server URL was not specified.\n";
            print_usage();
            exit 1;
        }
    }
    if (length($pipeline_template)==0){
        print "ERROR: a pipeline template was not specified.\n";
        print_usage();
        exit 1;  
    }elsif (! -e $pipeline_template){
        print "ERROR: The pipeline template file [$pipeline_template] does not exist.\n";
        print_usage();
        exit 1;  
    }
    
    if (length($infile)>0) {  #input file is local, upload it (use case 2)
        if (! -e $infile){ 
            print "ERROR: The input genome file [$infile] does not exist.\n";
            print_usage();
            exit 1;  
        }
        
        #upload input to shock
        print "uploading input file to Shock...\n";

        my $out_shock_upload = `curl -X POST -F upload=\@$infile $shock_url/node  | python -mjson.tool | grep \\\"id\\\"`;
        chomp($out_shock_upload);
        my @values = split('"', $out_shock_upload);
        $shock_id = $values[3];
                   
        if($? != 0) {
            print "Error: Failed to upload input file to Shock server, return value $?\n";
            exit $?;
        }
        print "\nuploading input file to Shock...Done! shock node id=".$shock_id."\n";
    } else { #input file is already on shock and the node id is specified by -node (use case 1)
        $shock_id = $node_id
    }
    
    #generate job script based on template (instantiate a job script with availalbe information filled into the template)
    my $pipeline_name = `basename $pipeline_template .template`;
    chomp($pipeline_name);
    print "pipeline_name=".$pipeline_name."\n";
    if (length($job_name)==0) {
        $job_name = $pipeline_name."-job";
    }
    print "job_name=".$job_name."\n";

    $jobscript = "tempjob.json";
        
    system("cp $pipeline_template $jobscript");
    system("perl -p -i -e 's/#shockurl/$shock_url/g;' $jobscript");
    system("perl -p -i -e 's/#shocknode/$shock_id/g;' $jobscript");
    system("perl -p -i -e 's/#project/$project_name/g;' $jobscript");
    system("perl -p -i -e 's/#user/$user_name/g;' $jobscript");
    system("perl -p -i -e 's/#jobname/$job_name/g;' $jobscript");
    system("perl -p -i -e 's/#clientgroups/$clients/g;' $jobscript");
    system("perl -p -i -e 's/#totalwork/$total_work/g;' $jobscript");
    
    
    
    if (length($infile_base)>0) {
        system("perl -p -i -e 's/#inputfile/$infile_base/g;' $jobscript");
    } else {
        system("perl -p -i -e 's/#inputfile/$job_name.$file_type/g;' $jobscript");
    }
    
} else { #Use case 3, use the already instantiated job script
    $jobscript = $input_script;
}

#upload job script to awe server
system("stat $jobscript");
print "submitting job script to AWE...jobscript=$jobscript \n";
my $out_awe_sub = `curl -X POST -F upload=\@$jobscript http://$awe_url/job | python -mjson.tool |  grep \\\"id\\\"`;
if($? != 0) {
    print "Error: Failed to submit job script to AWE server, return value $?\n";
    exit $?;
}
chomp($out_awe_sub);
my @values = split('"', $out_awe_sub);
my $job_id = $values[3];


print "\nsubmitting job script to AWE...Done! id=".$job_id."\n";

#print summary

print "job submission summary:\n";
print "pipeline job awe url: http://".$awe_url."/job/".$job_id."\n";

if (length($infile)>0 || length($node_id)>0) {
    print "input file shock url: http://".$shock_url."/node/".$shock_id."\n";
    my $refjson = "";
    if (length($infile)>0) {
       $refjson = "awe_".$infile_base."_".$job_id.".json"; 
    } else {
       $refjson = "awe_".$job_id.".json"; 
    }
    system("mv $jobscript $refjson");
    print "job script for reference: $refjson\n";
} 

exit(0);

sub print_usage{
    print "
Pipeline job submitter for AWE
Command name: awe_submit.pl
Options:
     -awe=<AWE server URL (ip:port), required>
     -shock=<Shock URL (ip:port), required>
     -node=<shock node of the input file>
     -type=<input file type, fna|fasta|fastq|fa, required when -node is set>
     -upload=<input file that is local and to be uploaded>
     -pipeline=<path for pipeline job template, required when -node or -upload is set>
     -script=<path for complete job json file>
     -name=<job name>
     -user=<user name>
     -project=<project name>
     -cgroups=<exclusive_client_group_list (separate by ',')>
     -totalwork=<number of workunits to split for splitable tasks (default 1)>
     
     
Use case 1: submit a job with a shock url for the input file location and a pipeline template (input file is on shock)
      Required options: -node, -pipeline, -awe (if AWE_HOST not in ENV), -shock (if SHOCK_HOST not in ENV)
      Optional options: -name, -user, -project, -cgroups, -totalwork
      Operations:
               1. create job script based on job template and available info
               2. submit the job json script to awe

Use case 2: submit a job with a local input file and a pipeline template (input file is local and will be uploaded to shock automatially;
      Required options: -upload, -pipeline, -awe (if AWE_HOST not in ENV), -shock (if SHOCK_HOST not in ENV)
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
