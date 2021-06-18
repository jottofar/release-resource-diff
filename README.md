# release-resource-diff

# Description
release-resource-diff checks if OpenShift Container Platform (OCP) resources, as defined by the manifests for one or more OCP releases, exist in a given newly installed OCP release. The purpose is to produce a list of resources which could exist on an upgraded clusters but can and should be removed.

# Usage

$ release-resource-diff -h  
Usage: release-resource-diff [-o \<results file path\>] [-v] \<target release file path\> \<top-level dir\>  
&nbsp;&nbsp;&nbsp;&nbsp;-o string  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;results file  
&nbsp;&nbsp;&nbsp;&nbsp;-v	verbose logging  

**"target release file path"** is a file containing all the resources from a running, newly installed, OCP release. This file is created by connecting to an OCP cluster running the OCP release to be compared against. The cluster must have been newly installed with the release, not upgraded, in order to get only the set of resources created by that specific release. Once connected to the cluster the file can be created by running "tools/create-target-release-file.sh TARGET_FILE_NAME" where  TARGET_FILE_NAME is the path to the file to be created. This file will have 4 columns for each resource: APIVersion, Kind, Name, Namespace. The release-resource-diff program depends on this file having these 4 coulmns in the given order. The file will be used to compare the resources from other OCP releases against to verify whether the resource still exists. For an example see test/4.9.0-0.nightly-2021-06-17-125213-all-resources.txt.
  
**"top-level dir"** is the path to a directory containing subdirectories for each OCP release to be checked. The subdirectories should be named using the release version number of its contents. For example, if the top-level directory were /tmp/releases it may contain subdirectories:
  
- 4.1.41
- 4.2.36
- ...
- 4.8.0-rc.0
  
These subdirectory names are used to fill in the "Born In" column of the results file. The results file also has a "Last In" column which indicates the latest release, of the given set being checked, in which the resource does exist. For example, if 4.2.36 resource R was not found in the target release but does exist in 4.8.0-rc.0 "Born In" would be 4.2.36 and "Last In" would be 4.8.0-rc.0. If the release numbering scheme is not used the program will not be able to determine "Last In" and it will simply be empty.

Each subdirectory is then populated with the manifests from that release by running **"oc adm release extract"**.

The program produces a file containing the results. By default the file is **"\<top-level dir\>/delete-candidates.txt"**. Use the "-o" option to override the default. The file is not display friendly so it is recommended that it be opened in a spreadsheet program such as LibreOffice Calc for ease of viewing and to allow sorting. test/delete-candidates.txt is an example results file.
