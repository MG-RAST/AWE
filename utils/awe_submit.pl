#!/usr/bin/env perl 

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
my $user_name = "default";
my $project_name = "default";
my $options = GetOptions ("input=s"   => \$input_file,
                          "awe=s"    => \$awe_url,
 			  "shock=s"  => \$shock_url,
			  "pipeline=s" => \$pipeline_template,
                          "user=s"   => \$user_name,
                          "project=s" => \$project_name,
			 );

if (length($awe_url)==0 ) {
    print "ERROR: AWE server URL was not specified.\n";
    print_usage();
    exit 1;
}

if (length($shock_url)==0 ) {
    print "ERROR: Shock server URL was not specified.\n";
    print_usage();
    exit 1;
}

if (length($input_file)==0){
    print "ERROR: An input file was not specified.\n";
    print_usage();
    exit 1;  #use line number as exit code
}elsif (! -e $input_file){
    print "ERROR: The input genome file [$input_file] does not exist.\n";
    print_usage();
    exit 1;  
}

if (length($pipeline_template)==0){
    print "ERROR: a pipeline template was not specified.\n";
    print_usage();
    exit 1;  #use line number as exit code
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
my $shock_id = $values[3];
                   
if($? != 0) {
    print "Error: Failed to upload input file to Shock server, return value $?\n";
    exit $?;
}
print "\nuploading input file to Shock...Done! shock node id=".$shock_id."\n";

#generate job script based on template
my $jobscript = "tempjob.json";

system("cp $pipeline_template $jobscript");
system("perl -p -i -e 's/#shockurl/$shock_url/g;' $jobscript");
system("perl -p -i -e 's/#shocknode/$shock_id/g;' $jobscript");
system("perl -p -i -e 's/#project/$project_name/g;' $jobscript");
system("perl -p -i -e 's/#user/$user_name/g;' $jobscript");

#upload job script to awe server

print "submitting job script to AWE...\n";
my $out_awe_sub = `curl -X POST -F upload=\@$jobscript http://$awe_url/job | python -mjson.tool |  grep \\\"id\\\"`;
chomp($out_shock_upload);
@values = split('"', $out_awe_sub);
my $job_id = $values[3];
if($? != 0) {
    print "Error: Failed to submit job script to AWE server, return value $?\n";
    exit $?;
}

print "\nsubmitting job script to AWE...Done! Job id=".$job_id."\n";

system("mv $jobscript awe_$job_id.json");

#print summary

print "job submission summary:\n";
print "input file shock url: http//".$shock_url."/node/".$shock_id."\n";
print "pipelien job awe url: http//".$awe_url."/job/".$job_id."\n";
print "job script for reference: awe_$job_id.json\n";


exit(0);

sub print_usage{
    print "USAGE: awe_submit.pl -input=<input file path> -awe=<AWE server URL (ip:port)> -shock=<Shock URL (ip:port)> -pipeline=<pipeline template path> [-user=<user name> -project=<project_name>]\n";
}
