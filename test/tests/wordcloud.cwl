#!/usr/bin/env cwl-runner
cwlVersion: v1.0

# Type of definition
#   CommandLineTool , Workflow , ExpressionTool


class:  CommandLineTool


# optional label
label: Word-Cloud

# optional description/documentation
# doc: <DETAILED DESCRIPTION>
# optional hints for CWL execution

hints:
# set execution environment for baseCommand
- class: DockerRequirement
  dockerPull: mgrast/pdf2wordcloud:demo


# required, name of command line tool
baseCommand: wordcloud_cli.py

# optional
# arguments: <LIST OF CONSTANT OR DERIVED COMMAND LINE OPTIONS>

# required, input mapping
inputs:
  text:
    type: File
    doc: input file to create wordcloud image from
    inputBinding:
      prefix: --text
  outname:
    type: string
    doc: Name for text output file
    inputBinding:
      prefix: --imagefile

# output mapping
outputs:
  image:
    type: File
    outputBinding:
      glob: $(inputs.outname)
