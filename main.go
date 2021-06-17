package main

import (
	"bufio"
	"bytes"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	Release      string
	YamlFileName string
}

var targetResources = make(map[ResourceId]bool)

var checkIt bool

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		usage(args)
		os.Exit(1)
	}
	loadTargetResources(args[0])

	releaseDirs := loadYamlFileDirs(args[1])

	if len(releaseDirs) == 0 {
		log.Fatalf("No directories found under %s", args[1])
	}
	fmt.Println("Checking...")

	deleteCandidates := make(map[ResourceId]ResourceSource)
	for _, dir := range releaseDirs {
		fmt.Printf("%s\n", dir)
		resources := getReleaseResources(args[1] + "/" + dir)
		deleteCandidates = checkIfOrphaned(resources, deleteCandidates)
	}
	//fmt.Printf("Candidate resources for deletion:\n%v\n", deleteCandidates)

	outputDeleteCandidates(args[1], deleteCandidates)
}

func usage(args []string) {
	if len(args) == 0 {
		fmt.Println("Expected arguments \"<target release file path> <top-level dir>\"")
	} else {
		fmt.Printf("Expected arguments \"<target release file path> <top-level dir>\"; got \"%v\"\n", args)
	}
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
			fmt.Printf("The target release file should have 4 columns: group, kind, name, namespace. Found %s\n", eachline)
			os.Exit(1)
		}
		resourceId := ResourceId{
			Group:     ids[0],
			Kind:      ids[1],
			Name:      ids[2],
			Namespace: ids[3],
		}
		targetResources[resourceId] = true
		//fmt.Printf("%v\n", resourceId)
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

func logIt(s string) {
	if checkIt {
		fmt.Println(s)
	}
}

func getReleaseResources(dir string) map[ResourceId]ResourceSource {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Unable to read dir %s; err=%v", dir, err)
	}

	base := filepath.Base(dir)
	ids := make(map[ResourceId]ResourceSource)

	for _, file := range files {
		info, _ := file.Info()

		checkIt = info.Name() == "0000_50_cluster-ingress-operator_00-ingress-credentials-request.yaml"

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
			Release:      base,
			YamlFileName: info.Name(),
		}
		var yamlId ResourceIdYaml

		for _, byteSlice := range allByteSlices {
			err = yaml.Unmarshal(byteSlice, &yamlId)
			if err != nil {
				log.Fatalf("Unable to unmarshall yaml from file %s; err=%v", filename, err)
			}
			if !validKey(yamlId) {
				log.Printf("Ignoring invalid resource key %v\n", yamlId)
				continue
			}
			if len(yamlId.Metadata.Namespace) == 0 {
				yamlId.Metadata.Namespace = "<none>"
			}
			resourceId := ResourceId{
				Group:     yamlId.APIVersion,
				Kind:      yamlId.Kind,
				Name:      yamlId.Metadata.Name,
				Namespace: yamlId.Metadata.Namespace,
			}
			ids[resourceId] = resourceSource
			//ids = append(ids, resourceId)
			//fmt.Printf("%v\n", yamlId)
		}
	}
	return ids
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

func checkIfOrphaned(resourceIds map[ResourceId]ResourceSource, currentOrphaned map[ResourceId]ResourceSource) map[ResourceId]ResourceSource {
	for k, v := range resourceIds {
		if _, ok := currentOrphaned[k]; !ok {
			if _, ok = targetResources[k]; !ok {
				currentOrphaned[k] = v
			}
		}
	}
	return currentOrphaned
}

func outputDeleteCandidates(dir string, resources map[ResourceId]ResourceSource) {
	filePath := dir + "/delete-candidates.txt"
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Unable to create file %s; err=%v", filePath, err)
	}
	defer f.Close()
	for k, v := range resources {
		s := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n",
			k.Group, k.Kind, k.Name, k.Namespace, v.Release, v.YamlFileName)
		_, err := f.WriteString(s)
		if err != nil {
			log.Fatalf("Unable to write to file %s; err=%v", filePath, err)
		}
	}
}
