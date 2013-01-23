#!/usr/bin/env perl 
# job submitter for AWE
# Command name: awe_submit.pl
# Options:
#     -awe=<AWE server URL (ip:port)>
#     -input=<input file path>
#     -shock=<Shock URL (ip:port)>
#     -pipeline=<path for pipeline job template>
#     -script=<path for complete job json file>
#     -user=<user name>
#     -project=<project_name>
#     -clients=<exclusive_client_name_list (separate by ",")>
#
# Use case 1: submit a job with a input file and a pipeline template (input file is local, suitable for first-time submission).
#      Required options: -input, -awe, -shock, -pipeline
#      Optional options: -user, -project
#      Operations:
#               1. upload input file to shock
#               2. fill shock url into the pipeline template and make a job json script
#               3. submit the job json script to awe
#        
# UUse case 2: submit a job with a complete job json script (input file is already in shock, suitable for recomputation)
#      Required options: -script, -awe
#      Optional options: none (all needed info is in the job script)
#      Operations: submit the job json script to awe directly 

use strict;
use warnings;
no warnings('once');

use Getopt::Long;
use File::Copy;
use File::Basename;
use POSIX qw(strftime);
umask 000;

# options
my $input_file = "";
my $awe_url = "";
my $shock_url = "";
my $pipeline_template = "";
my $input_script = "";
my $user_name = "default";
my $project_name = "default";
my $clients="";
my $help = 0;

my $options = GetOptions ("input=s"   => \$input_file,
                          "awe=s"    => \$awe_url,
 			  "shock=s"  => \$shock_url,
			  "pipeline=s" => \$pipeline_template,
                          "script=s"   => \$input_script,
                          "user=s"   => \$user_name,
                          "project=s" => \$project_name,
                          "clients=s" => \$clients,
                          "h"  => \$help,
			 );

my $jobscript = "";
my $shock_id = "";

if ($help) {
    print_usage();
    exit 0;
}

if (length($awe_url)==0 ) {
    print "ERROR: AWE server URL was not specified.\n";
    print_usage();
    exit 1;
}

if (length($input_file)>0 && length($input_script)>0) {
    print "ERROR: input file and json script are mutual exclusive, specify only one of them\n";
    print_usage();
    exit 1;
}

if (length($input_file)==0 && length($input_script)==0) {
    print "ERROR: please specify the path of either input file or the job json script\n";
    print_usage();
    exit 1;
}

if (length($input_file)>0) { #Use case 1
    if (! -e $input_file){
        print "ERROR: The input genome file [$input_file] does not exist.\n";
        print_usage();
        exit 1;  
    }
    if (length($shock_url)==0 ) {
        print "ERROR: Shock server URL was not specified.\n";
        print_usage();
        exit 1;
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

    #upload input to shock
    print "uploading input file to Shock...\n";

    my $out_shock_upload = `curl -X POST -F upload=\@$input_file $shock_url/node  | python -mjson.tool | grep \\\"id\\\"`;
    chomp($out_shock_upload);
    my @values = split('"', $out_shock_upload);
    $shock_id = $values[3];
                   
    if($? != 0) {
        print "Error: Failed to upload input file to Shock server, return value $?\n";
        exit $?;
    }
    print "\nuploading input file to Shock...Done! shock node id=".$shock_id."\n";

    #generate job script based on template
    $jobscript = "tempjob.json";

    system("cp $pipeline_template $jobscript");
    system("perl -p -i -e 's/#shockurl/$shock_url/g;' $jobscript");
    system("perl -p -i -e 's/#shocknode/$shock_id/g;' $jobscript");
    system("perl -p -i -e 's/#project/$project_name/g;' $jobscript");
    system("perl -p -i -e 's/#user/$user_name/g;' $jobscript");
    system("perl -p -i -e 's/#clients/$clients/g;' $jobscript");
} else { #Use case 2
    $jobscript = $input_script;
}

#upload job script to awe server

print "submitting job script to AWE...\n";
my $out_awe_sub = `curl -X POST -F upload=\@$jobscript http://$awe_url/job | python -mjson.tool |  grep \\\"id\\\"`;
chomp($out_awe_sub);
my @values = split('"', $out_awe_sub);
my $job_id = $values[3];
if($? != 0) {
    print "Error: Failed to submit job script to AWE server, return value $?\n";
    exit $?;
}

print "\nsubmitting job script to AWE...Done! Job id=".$job_id."\n";

#print summary

print "job submission summary:\n";
print "pipeline job awe url: http://".$awe_url."/job/".$job_id."\n";
if (length($input_file)>0) {
    print "input file shock url: http://".$shock_url."/node/".$shock_id."\n";
    system("mv $jobscript awe_$job_id.json");
    print "job script for reference: awe_$job_id.json\n";
}

exit(0);

sub print_usage{
    print "Pipeline job submitter for AWE\n";
    print "Command name: awe_submit.pl\n";
    print "Options:\n";
    print "     -awe=<AWE server URL (ip:port)>\n";
    print "     -input=<input file path>\n";
    print "     -shock=<Shock URL (ip:port)>\n";
    print "     -pipeline=<path for pipeline job template>\n";
    print "     -script=<path for complete job json file>\n";
    print "     -user=<user name>\n";
    print "     -project=<project_name>\n";
    print "     -clients=<exclusive_client_name_list (separate by ',')\n";
    print "\n";
    print "Use case 1: submit a job with a input file and a pipeline template (input file is local, suitable for first-time submission)\n";
    print "      Required options: -input, -awe, -shock, -pipeline\n";
    print "      Optional options: -user, -project, -clients\n";
    print "      Operations:\n";
    print "               1. upload input file to shock\n";
    print "               2. fill shock url into the pipeline template and make a job json script\n";
    print "               3. submit the job json script to awe\n";
    print "\n";
    print "Use case 2: submit a job with a complete job json script (input file is already in shock, suitable for recomputation)\n";
    print "      Required options: -script, -awe\n";
    print "      Optional options: none  (all needed info is in the job script)\n";
    print "      Operations: submit the job json script to awe directly.\n";
}