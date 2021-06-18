package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type ResourceIdYaml struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	}
}

type ResourceId struct {
	Group     string
	Kind      string
	Name      string
	Namespace string
}

type ResourceSource struct {
	Release       string
	LastInRelease string
	YamlFileName  string
}

var (
	targetResources = make(map[ResourceId]bool)

	verboseLogging bool

	resultsFile string
)

func main() {
	flag.BoolVar(&verboseLogging, "v", false, "verbose logging")
	flag.StringVar(&resultsFile, "o", "", "results file")
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		usage()
		os.Exit(1)
	}
	if len(resultsFile) == 0 {
		resultsFile = args[1] + "/delete-candidates.txt"
	}
	loadTargetResources(args[0])

	releaseDirs := loadYamlFileDirs(args[1])

	if len(releaseDirs) == 0 {
		log.Fatalf("No directories found under %s", args[1])
	}
	fmt.Println("Checking...")

	deleteCandidates := make(map[ResourceId]ResourceSource)
	for _, dir := range releaseDirs {
		fmt.Printf("%s: ", dir)
		resources := getReleaseResources(args[1] + "/" + dir)
		fmt.Printf("%d resources checked\n", len(resources))
		checkIfOrphaned(resources, deleteCandidates)
	}
	outputDeleteCandidates(deleteCandidates)
}

func usage() {
	fmt.Printf("Usage: %s [-o <results file path>] [-v] <target release file path> <top-level dir>\n", os.Args[0])
	flag.PrintDefaults()
}

func loadTargetResources(path string) {
	target, _ := filepath.Abs(path)
	file, err := os.Open(target)
	if err != nil {
		log.Fatalf("Unable to read target release file %s; err=%v", path, err)
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var txtlines []string

	for scanner.Scan() {
		txtlines = append(txtlines, scanner.Text())
	}

	file.Close()

	for _, eachline := range txtlines {
		ids := strings.Fields(eachline)
		if len(ids) != 4 {
			log.Fatalf("The target release file should have 4 columns: group, kind, name, namespace. Found %s\n", eachline)
		}
		resourceId := ResourceId{
			Group:     truncateVersion(ids[0]),
			Kind:      ids[1],
			Name:      ids[2],
			Namespace: ids[3],
		}
		targetResources[resourceId] = true
		logIt(fmt.Sprintf("%v", resourceId))
	}
}

func loadYamlFileDirs(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Unable to read dir %s; err=%v", dir, err)
	}

	var dirs []string
	for _, file := range files {
		info, _ := file.Info()
		if info.IsDir() {
			dirs = append(dirs, info.Name())
		}
	}
	return dirs
}

func getReleaseResources(dir string) map[ResourceId]ResourceSource {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Unable to read dir %s; err=%v", dir, err)
	}

	minor := getMinorRelease(filepath.Base(dir))
	ids := make(map[ResourceId]ResourceSource)

	for _, file := range files {
		info, _ := file.Info()

		if filepath.Ext(info.Name()) != ".yaml" {
			continue
		}
		filename := dir + "/" + info.Name()
		yamlBytes, err := ioutil.ReadFile(filename)

		if err != nil {
			log.Fatalf("Unable to read file %s; err=%v", filename, err)
		}

		allByteSlices, err := splitYaml(yamlBytes)
		if err != nil {
			log.Fatalf("Unable to split yaml from file %s; err=%v", filename, err)
		}

		resourceSource := ResourceSource{
			Release:       minor,
			LastInRelease: minor,
			YamlFileName:  info.Name(),
		}
		logIt(fmt.Sprintf("%v", resourceSource))
		var yamlId ResourceIdYaml

		for _, byteSlice := range allByteSlices {
			err = yaml.Unmarshal(byteSlice, &yamlId)
			if err != nil {
				log.Fatalf("Unable to unmarshall yaml from file %s; err=%v", filename, err)
			}
			if !validKey(yamlId) {
				logIt(fmt.Sprintf("Ignoring invalid resource key %v", yamlId))
				continue
			}
			if len(yamlId.Metadata.Namespace) == 0 {
				yamlId.Metadata.Namespace = "<none>"
			}
			logIt(fmt.Sprintf("%v", yamlId))
			resourceId := ResourceId{
				Group:     truncateVersion(yamlId.APIVersion),
				Kind:      yamlId.Kind,
				Name:      yamlId.Metadata.Name,
				Namespace: yamlId.Metadata.Namespace,
			}
			ids[resourceId] = resourceSource
		}
	}
	return ids
}

func logIt(s string) {
	if verboseLogging {
		fmt.Println(s)
	}
}

func truncateVersion(apiVersion string) string {
	api := strings.Split(apiVersion, "/")
	if len(api) != 2 {
		return apiVersion
	}
	return api[0]
}

func getMinorRelease(release string) string {
	xyz := strings.Split(release, ".")
	if len(xyz) < 2 {
		return release
	}
	return xyz[0] + "." + xyz[1]
}

func splitYaml(resources []byte) ([][]byte, error) {

	dec := yaml.NewDecoder(bytes.NewReader(resources))

	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}
	return res, nil
}

func validKey(key ResourceIdYaml) bool {
	if len(key.APIVersion) == 0 || len(key.Kind) == 0 ||
		len(key.Metadata.Name) == 0 {
		return false
	}
	return true
}

func checkIfOrphaned(resourceIds map[ResourceId]ResourceSource, currentOrphaned map[ResourceId]ResourceSource) {
	for k, v := range resourceIds {
		if _, ok := currentOrphaned[k]; !ok {
			if _, ok = targetResources[k]; !ok {
				currentOrphaned[k] = v
			}
		} else {
			setLastInRelease(v.LastInRelease, currentOrphaned, k)
		}
	}
}

func setLastInRelease(thisRelease string, currentOrphaned map[ResourceId]ResourceSource, key ResourceId) {
	if thisRelVal, err := strconv.ParseFloat(thisRelease, 32); err == nil {
		if val, err := strconv.ParseFloat(currentOrphaned[key].LastInRelease, 32); err == nil {
			if thisRelVal > val {
				currentOrphaned[key] = ResourceSource{
					Release:       currentOrphaned[key].Release,
					LastInRelease: thisRelease,
					YamlFileName:  currentOrphaned[key].YamlFileName,
				}
			}
		}
	}
}

func outputDeleteCandidates(resources map[ResourceId]ResourceSource) {
	f, err := os.Create(resultsFile)
	if err != nil {
		log.Fatalf("Unable to create file %s; err=%v", resultsFile, err)
	}
	defer f.Close()
	for k, v := range resources {
		s := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			k.Group, k.Kind, k.Name, k.Namespace, v.Release, v.LastInRelease, v.YamlFileName)
		_, err := f.WriteString(s)
		if err != nil {
			log.Fatalf("Unable to write to file %s; err=%v", resultsFile, err)
		}
	}
	fmt.Printf("Results file %s created\n", resultsFile)
}
