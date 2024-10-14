package template

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

//go:generate go run actor_template_gen.go

type ActorConfig struct {
	Name     string            `yaml:"name"`
	Unique   bool              `yaml:"unique"`
	Category string            `yaml:"category"`
	Options  map[string]string `yaml:"options,omitempty"`
}

type NodeConfig struct {
	ID        string                  `yaml:"id"`
	Weight    string                  `yaml:"weight"`
	ActorOpts []RegisteredActorConfig `yaml:"actors"`
}

type Config struct {
	Node NodeConfig `yaml:"node"`
}

type RegisteredActorConfig struct {
	Name    string            `yaml:"name"`
	Options map[string]string `yaml:"options,omitempty"`
}

type ActorTypes struct {
	ActorTypes []ActorConfig `yaml:"actor_templates"`
}

func loadYAML(filename string, v interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}

func ParseConfig(confPath, actorTypesPath string) (*NodeConfig, []ActorConfig, error) {
	// 读取配置文件
	configData, err := ioutil.ReadFile(confPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read main config: %v", err)
	}

	// 读取 actor 类型文件
	actorTypesData, err := ioutil.ReadFile(actorTypesPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read actor types: %v", err)
	}

	return ParseConfigFromString(string(configData), string(actorTypesData))
}

func ParseConfigFromString(confData, actorTypesData string) (*NodeConfig, []ActorConfig, error) {

	// 解析 actor 类型
	var actorTypes ActorTypes
	if err := yaml.Unmarshal([]byte(actorTypesData), &actorTypes); err != nil {
		return nil, nil, fmt.Errorf("failed to parse actor types: %v", err)
	}

	// 创建 actor 类型映射，用于快速查找
	actorTypeMap := make(map[string]ActorConfig)
	for _, actorType := range actorTypes.ActorTypes {
		actorTypeMap[actorType.Name] = actorType
	}

	// 解析主配置
	var config Config
	if err := yaml.Unmarshal([]byte(confData), &config); err != nil {
		return nil, nil, fmt.Errorf("failed to parse main config: %v", err)
	}

	// 解析节点配置
	nodeID := os.Getenv("BRAID_NODE_ID")
	if nodeID == "" {
		nodeID = config.Node.ID
	}
	nodeWeight := os.Getenv("BRAID_NODE_WEIGHT")
	if nodeWeight == "" {
		nodeWeight = config.Node.Weight
	}

	var parsedActors []ActorConfig
	for _, registeredActor := range config.Node.ActorOpts {
		actorType, ok := actorTypeMap[registeredActor.Name]
		if !ok {
			return nil, nil, fmt.Errorf("actor %s is registered in node.yml but not defined in actor_template.yml", registeredActor.Name)
		}

		actor := ActorConfig{
			Name:     actorType.Name,
			Unique:   actorType.Unique,
			Category: actorType.Category,
			Options:  registeredActor.Options, // 使用 conf.yml 中的 options
		}
		parsedActors = append(parsedActors, actor)
	}

	nodeConfig := &NodeConfig{
		ID:        nodeID,
		Weight:    nodeWeight,
		ActorOpts: config.Node.ActorOpts,
	}

	return nodeConfig, actorTypes.ActorTypes, nil
}
