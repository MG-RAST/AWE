#!/usr/bin/env sh
# usage: bowtie_screen.sh <screening_idexes> <input_fasta> <screened_output_fasta> [<threads>]
# VERSION: $Id$
# purpose: run bowtie to filter for some organism (eg. human)
# required: environment varible $REFDBPATH is configured as a valid path under which the index files can be found

if [ $# -lt 2 ] ; then
      echo "Usage: bowtie_screen.sh <screening_idexes> <input_fa> <output_fa> [<threads>]"
      exit 0
fi

# optional number of threads
threads=${4-1}

# if environment varible $REFDBPATH is configured as a valid path under which the index files can be found

if [ ${REFDBPATH} ]
then INDEX=$REFDBPATH/$1
else INDEX=$1
fi

INPUT=`basename $2`
OUTPUT=`basename $3`
echo INPUT $INPUT > REPORT
echo INDEX $INDEX >> REPORT
echo SCREENED_OUTPUT $3 >> REPORT
echo CALL bowtie --suppress 5,6 -p ${threads}  --al aligned_reads.fasta --un $OUTPUT -f -t $INDEX $INPUT bowtie.out >> REPORT
bowtie --suppress 5,6 -p ${threads}  --al aligned_reads.fasta --un $OUTPUT -f -t $INDEX $INPUT bowtie.out
