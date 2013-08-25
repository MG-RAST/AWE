#!/usr/bin/env perl 
# show awe_server queue status
# Command name: awe_qstat.pl
# Options:
#     -awe=<AWE server URL (ip:port)> (required)
#     -r   (show status repeatedly every 10 seconds)

use strict;
use warnings;
no warnings('once');

use Getopt::Long;
use File::Copy;
use File::Basename;
use POSIX qw(strftime);
umask 000;

# options
my $awe_url = "";
my $help = 0;
my $repeat = 0;

my $options = GetOptions ("awe=s"    => \$awe_url,
			  "r" => \$repeat,
                          "h"  => \$help,
			 );


if ($help) {
    print_usage();
    exit 0;
}

while (1) {
    my $cmd = 'curl -s -S -X GET "'.$awe_url.'/queue" | cut -d "\"" -f 6';
    my $msg = `$cmd`;
    my @values=split(/\\n/,$msg);
    foreach  my $val (@values) {
        print $val . "\n";
    }
    if ($repeat==0) {
	exit 0;
    }
    sleep(10);
} 

sub print_usage{
  print "show awe_server queue status
  Command name: awe_qstat.pl
  Options:
     -awe=<AWE server URL (ip:port)> (required)
     -r   (show status repeatedly every 10 seconds)\n"
}