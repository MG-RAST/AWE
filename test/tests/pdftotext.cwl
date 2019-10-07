#!/usr/bin/env cwl-runner
cwlVersion: v1.0

# Type of definition
#   CommandLineTool , Workflow , ExpressionTool

class:  CommandLineTool

# optional label
label: PDF-to-Text

# optional description/documentation
# doc: <DETAILED DESCRIPTION>

# optional hints for CWL execution
hints:
# set execution environment for baseCommand
- class: DockerRequirement
  dockerPull: mgrast/pdf2wordcloud:demo

# required, name of command line tool
baseCommand: pdftotext

# optional
# arguments: <LIST OF CONSTANT OR DERIVED COMMAND LINE OPTIONS>

# required, input mapping
inputs:
  pdf:
    type: File
    doc: PDF input file to extract text from
    inputBinding:
      position: 1
  text:
    type: string
    doc: Name for text output file
    inputBinding:
      position: 2

# output mapping
outputs:
  extractedText:
    type: File
    outputBinding:
      glob: $(inputs.text)
