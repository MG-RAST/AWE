Skyport app definitions make it possible to use well defined functions instead of command line syntax. This abstraction layer makes it easier to reuse tools and also simplifies connection of outputs and inputs from different tools to create novel scientific workflows. 

App definitions are organized in packages, representing the set of all functions provided by one Linux container. Each app definition file represents one package. For example [Bowtie2.json](https://github.com/MG-RAST/Skyport/blob/master/app_definitions/Bowtie2.json) is a package that contains Bowtie2 functions to build a Bowtie index and to do a Bowtie search. Function names are composed of the package name (corresponds to the filename without .json), the tool name and a tool mode specifier. In the example of Bowtie2 the function names are "Bowtie2.bowtie2-build.default" and "Bowtie2.bowtie2.default". The lowest level in the naming scheme, the tool mode specifier, makes it possible to use different app definitions for the same command line tool. This can be useful if the syntax of the different modes is considerably different. It can also be used to provide different default parameter settings for the same tool. The Bowtie example above does not make use of this functionality, thus the keyword "default" is used. Packages can be organized in sub directories. If package "mypackage" is saved in file mydir/mypackage.json, the package name will be "mydir/mypackage".

An app definition package specifies the docker image ("dockerimage") and the functions ("commands") that are defined: 
```text
{
    "dockerimage" : "skyport/bowtie2:2.1.0",
    "commands"  : {
        ... app definitions ...
    }
}
```

## "dockerimage" ##
The "dockerimage" field identifies the docker image stored in the Shock server that contains all tools. Please see ... for documentation on how to store docker images in Shock. The "dockerimage" identifier is used by the AWE workers to download docker images from Shock. It has to have the format "REPO/NAME:VERSIONTAG". "REPO" could be something like a user name, "NAME" should be a tool or package name and "VERSIONTAG" a version number preferably using the version string format specified at the end of this document. VERSIONTAG can either be a specific version number or the keyword "latest", for example, "skyport/bowtie2:latest". In that case the AWE worker will try to find the image with the highest version number. Using the "latest" keyword is convenient, e.g. for development, but is potentially less stable and does not guarantee reproducibility. For production workflow app definitions it is recommended to use a specific version string for the docker image.  

## App defintions ##
Each app definition under "commands" specifies inputs, outputs and the actual command line needed to invoke the tool. The field "variables" provides an optional mechanism to define variables to be used in the command line or in output file names.

### Inputs ###
```text
"input" : [
    { 
    "type"	    : "file",
    "name"	    : "INPUT-FILE"
    },
    {
    "type"	    : "file",
    "name"	    : "INPUT-PARAMETER"
    },
    { 
    "type"	    : "string",
    "name"	    : "IDENTITY",
    "default_value" : "97",
    "description"   : "percent identity"
    },
    { 
    "type"	    : "string",
    "name"	    : "OUTPUT-DIR",
    "default_value" : "ucr",
    "description"   : "output directory"
    }
],
``` 
[For complete example see here](https://github.com/MG-RAST/Skyport/blob/master/app_definitions/QIIME.json)

Input is an array of "input resources". Each input resource has a type and a name. Other keys such as "default_value" and "description" are optional. Supported types currently are "file", "string" and "list". Type "file" refers to files in Shock and "string" are normal string arguments. Numeric arguments would currently also be specified as a string. The "list" type specifies a list of input resources, that allows an app function to accept a variable number of input files. Elements of a list are input resources, but cannot be a list type themselves.

```text
{
    "type"	: "list",
    "name"	: "index_files",
    "description" : "this are the bowtie index files .bt2"
}
```
Example of a resource input of type list. [Complete example](https://github.com/MG-RAST/Skyport/blob/master/app_definitions/MG-RAST/bowtie.json)

Each input resource creates variables that can be used like Shell environment variables. For example, the first input resource with the name "INPUT-FILE", generates the variable "${INPUT-FILE}" that evaluates to the local filesystem filename. In addition, appending suffixes .host, .node and .url to the variable name generates shock host URL, shock node identifier and the complete URL of the input resource. (Example: ${INPUT-FILE.host} gives URL of Shock server)

Files that are passed as an argument to an app are stored on the local filesystem of the AWE worker using the filename specified in the Shock node. This can be overwritten with "filename":
```text
{ 
    "type"     : "file",
    "name"     : "INPUT-FILE",
    "filename" : "fixed.dat"
}
```

### Variables ###

Default variables:

```text
${NumCPU}    Number of cores on machine
${datatoken} Shock authentication token (useful in combination with ".node")
${arguments} Uses all resources to automatically create --name=value  (in development)
```
Function variable:
```text
${remove_extension:FILENAME}   Removes extension (suffix) including the dot. Will remove only one dot.
Example A: ${remove_extension:helloworld.txt}   Produces "helloworld"
Example B: ${remove_extension:${INPUT}}.fasta   Remove any existing suffix and append ".fasta" 
```
Typical use case of "remove_extension" is the generation of output filenames that have the same prefix as the input filename. Two file extension can be removed with ${remove_extension:${remove_extension:FILENAME}}. If filename has no extension left, filename will not be changed.

Variable definition:

Variables can be defined in an array of collections of key/value pairs. The outer array ensures evaluation of variables in a specific order if needed. The inner collection does not guarantee a specific order.
```text
"variables" : [{ "PREFIX" : "${remove_extension:${INPUT}}.bowtie_db"}],
```
This example defines variable ${PREFIX}.


### Execution Script ###
"cmd_script" is an array of strings that are executed in an auto-generated bash script. 

```text
"variables" :
    [
        { 
            "DBPREFIX" : "${remove_extension:${remove_extension:${DB.1.bt2}}}",
            "OUTPUT" : "${remove_extension:${INPUT}}.sam"
        }
    ],
"cmd_script" : 
    [
        "bowtie2 -x ${DBPREFIX} -f ${INPUT} -S ${OUTPUT}"
    ],
```
Example: [Bowtie2.json](https://github.com/MG-RAST/Skyport/blob/master/app_definitions/Bowtie2.json)

At the moment all tools are invoked via such a bash script. This provides flexibility and can avoid the need of deploying a wrapper script onto the Docker image. While we think this is a useful feature, in particular when working with third-party images, it also means that each image must come with Bash installed. This is the case with standard base images such as ubuntu, but not necessarily with very minimalistic "scratch" images that consist only of statically compiled binaries. We will add the ability to support those images in the near future.

### Outputs ###

```text
"outputs" : [
    {
        "name": "assembly.coverage",
        "filename" : "${job_id}.075.assembly.coverage",
        "attrfile" : "${job_id}.075.assembly.coverage.json"
    },
    {
        "name": "qc.stats",
        "filename" : "${job_id}.075.qc.stats",
        "attrfile" : "${job_id}.075.qc.stats.json"
    },
    {
        "name": "upload.stats",
        "filename" : "${job_id}.075.upload.stats",
        "attrfile" : "${job_id}.075.upload.stats.json"
    }
]
```

Similar to inputs, each output can have an abstract name. These names are helpful for connecting inputs and outputs of tasks in a workflow. "filename" is the important information that the AWE worker needs to find and upload compute results. "attrfile" specifies metadata in JSON format for the new Shock node. This optional file has to be created by the tool itself, it is not created by AWE.



### Supported version string formats (TODO) ###

Examples:
```text
2.0.0 > 1.0.0
1.0.1 > 1.0.0
1.0.1 > 1.0.1b
20140203 > 20140202
beta > alpha
```


