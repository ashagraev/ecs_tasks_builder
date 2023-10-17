package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type FlagsArray []string

func (a *FlagsArray) String() string {
	return strings.Join(*a, " ")
}

func (a *FlagsArray) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func (a *FlagsArray) ToEnvironmentVariables() ([]*EnvironmentVariable, error) {
	var environmentVariables []*EnvironmentVariable
	for _, v := range *a {
		parts := strings.Split(v, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("environment variable %q must have form \"KEY=VALUE\"", v)
		}
		environmentVariables = append(environmentVariables, &EnvironmentVariable{
			Name:  parts[0],
			Value: parts[1],
		})
	}
	return environmentVariables, nil
}

type TaskDefinitionData map[string]interface{}

func ReadTaskDefinition(filePath string) (*TaskDefinitionData, error) {
	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	td := &TaskDefinitionData{}
	if err := json.Unmarshal(fileContents, &td); err != nil {
		return nil, err
	}
	return td, nil
}

func (td *TaskDefinitionData) ExtractSlice(key string, title string) ([]interface{}, error) {
	value, ok := (*td)[key]
	if !ok {
		return nil, fmt.Errorf("%s doesn't have a %q key", title, key)
	}
	switch value.(type) {
	case []interface{}:
	default:
		return nil, fmt.Errorf("%s has a key %q, but the value is not a slice", title, key)
	}
	return value.([]interface{}), nil
}

func extractMap(source interface{}, key string, title string) (map[string]interface{}, error) {
	switch source.(type) {
	case map[string]interface{}:
	default:
		return nil, fmt.Errorf("%s doesn't have a map type", title)
	}
	sourceMap := source.(map[string]interface{})
	value, ok := sourceMap[key]
	if !ok {
		return nil, fmt.Errorf("%s doesn't have a %q key", title, key)
	}
	switch value.(type) {
	case map[string]interface{}:
	default:
		return nil, fmt.Errorf("%s has a key %q, but the value is not a map", title, key)
	}
	return value.(map[string]interface{}), nil
}

func extractString(source interface{}, key string, title string) (string, error) {
	switch source.(type) {
	case map[string]interface{}:
	default:
		return "", fmt.Errorf("%s doesn't have a map type", title)
	}
	sourceMap := source.(map[string]interface{})
	value, ok := sourceMap[key]
	if !ok {
		return "", fmt.Errorf("%s doesn't have a %q key", title, key)
	}
	switch value.(type) {
	case string:
	default:
		return "", fmt.Errorf("%s has a key %q, but the value is not a string", title, key)
	}
	return value.(string), nil
}

func (td *TaskDefinitionData) ModifyContainerDefinition(containerName string, containerTag string, dataDogTags []string, environmentVariables []*EnvironmentVariable) error {
	containerDefinitions, err := td.ExtractSlice("containerDefinitions", "source task definition")
	if err != nil {
		return err
	}
	for idx, containerDefinition := range containerDefinitions {
		switch containerDefinition.(type) {
		case map[string]interface{}:
		default:
			return fmt.Errorf("container definition #%d doesn't have a map type", idx)
		}
		containerDefinitionMap := containerDefinition.(map[string]interface{})
		title := fmt.Sprintf("container definition #%d", idx)
		name, err := extractString(containerDefinitionMap, "name", title)
		if err != nil {
			return err
		}
		if name != containerName {
			continue
		}
		if containerTag != "" {
			imageName, err := extractString(containerDefinitionMap, "image", title)
			if err != nil {
				return err
			}
			imageNameParts := strings.Split(imageName, ":")
			if len(imageNameParts) > 2 {
				return fmt.Errorf("image name %q contains more than one colon", imageName)
			}
			containerDefinitionMap["image"] = imageNameParts[0] + ":" + containerTag
		}
		logConfiguration, err := extractMap(containerDefinitionMap, "logConfiguration", title)
		if err != nil {
			return err
		}
		options, err := extractMap(logConfiguration, "options", title)
		if err != nil {
			return err
		}
		options["dd_tags"] = strings.Join(dataDogTags, ",")
		containerDefinitionMap["environment"] = environmentVariables
	}
	return nil
}

func (td *TaskDefinitionData) MarshalToJSON() ([]byte, error) {
	jsonData, err := json.Marshal(td)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	if err := json.Indent(&b, jsonData, "", "  "); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func main() {
	dataDogTags := &FlagsArray{}
	flag.Var(dataDogTags, "dd_tag", "Data dog tag. Example: --dd_tag cloud:aws --dd_tag environment:production.")
	environmentVariables := &FlagsArray{}
	flag.Var(environmentVariables, "env", "Environment variable. Example: --env GUNICORN_WORKERS=4 --env OTHER_KEY=OTHER_VALUE.")
	containerName := flag.String("container", "", "Container name to work with.")
	containerTag := flag.String("tag", "", "Container tag to set up in the task definition.")
	sourceDefinitionPath := flag.String("source", "", "Source task definition file path.")
	outputDefinitionPath := flag.String("output", "", "Output task definition file path.")
	flag.Parse()

	parsedEnvironmentVariables, err := environmentVariables.ToEnvironmentVariables()
	if err != nil {
		log.Fatalln(err)
	}
	td, err := ReadTaskDefinition(*sourceDefinitionPath)
	if err != nil {
		log.Fatalln(err)
	}
	if err := td.ModifyContainerDefinition(*containerName, *containerTag, *dataDogTags, parsedEnvironmentVariables); err != nil {
		log.Fatalln(err)
	}
	jsonData, err := td.MarshalToJSON()
	if err != nil {
		log.Fatalln(err)
	}
	if *outputDefinitionPath == "" {
		fmt.Println(string(jsonData))
	} else {
		if err := os.WriteFile(*outputDefinitionPath, jsonData, 0o644); err != nil {
			log.Fatalln(err)
		}
	}
}
