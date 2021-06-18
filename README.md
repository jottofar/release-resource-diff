# release-resource-diff

# Description
Checks if OpenShift Container Platform (OCP) resources, as defined by the manifests for one or more OCP releases, exist in a given newly installed OCP release. The purpose is to produce a list of resources which could exist on an upgraded clusters but can and should be removed.

# Usage
+
[source,terminal]
----
$ release-resource-diff -h
Usage: release-resource-diff [-o <results file path>] [-v] <target release file path> <top-level dir>
  -o string
    	results file
  -v	verbose logging
----
+
**"target release file path"** is a file containing all the resources from a running, newly installed, OCP release. This file is created by connecting to an OCP cluster running the OCP release to be compared against. The cluster must have been newly installed with the release, not upgraded, in order to get only the set of resources created by that specific release. Once connected to the cluster the file can be created by running "tools/create-target-release-file.sh TARGET_FILE_NAME" where  TARGET_FILE_NAME is the path to the file to be created. This file will have 4 columns for each resource: APIVersion, Kind, Name, Namespace. The release-resource-diff program depends on this file having these 4 coulmns in the given order. The file will be used to compare the resources from other OCP releases against to verify whether the resource still exists.
  
**"top-level dir"** is the path to a directory containing subdirectories for each OCP release to be checked. The subdirectories should be named using the release version number of its contents. For example, if the top-level directory were /tmp/releases it may contain subdirectories:
  
- 4.1.41
- 4.2.36
  ...
- 4.8.0-rc.0
  
These subdirectory names are used to fill in the "Born In" column of the results file. The results file also has a "Last In" column which indicates the latest release, of the given set being checked, in which the resource does exist. For example, if 4.2.36 resource R was found not to exist in the target but does exist in 4.8.0-rc.0 "Born In" would be 4.2.36 and "Last In" would be 4.8.0-rc.0. If the release numbering scheme is not used the program will not be able to determine "Last In" and it will simply be empty.

Each subdirectory is then populated with th emanifests from that release by running
