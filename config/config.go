package config

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"reflect"
	"strings"

	"github.com/imdario/mergo"
)

type ReadFileDir interface {
	fs.ReadDirFS
	fs.ReadFileFS
}

type confStruct struct {
	profileActive     string
	confPrefix        string
	unmarshalType     UnmarshalType
	envName           string
	defaultEnvValue   string
	activeProfileFunc func() string
	masterConfRaw     []byte
	activeConfRaw     []byte
}

func Unmarshal(confDir ReadFileDir, v interface{}, ops ...Option) error {
	if v != nil && reflect.ValueOf(v).Kind() != reflect.Ptr {
		return ErrNonPointerArgument
	}

	var cs = &confStruct{
		confPrefix:      "conf",
		envName:         "profile",
		defaultEnvValue: "dev",
		unmarshalType:   Yaml,
	}
	cs.activeProfileFunc = func() string {
		return getActiveProfile(cs.envName, cs.defaultEnvValue)
	}

	for _, op := range ops {
		op(cs)
	}

	cs.profileActive = cs.activeProfileFunc()

	if err := loadConf(confDir, cs); err != nil {
		return err
	}

	// copy val
	v1 := reflect.New(reflect.TypeOf(v).Elem()).Interface()

	if err := fileUnmarshal(cs.activeConfRaw, v1, cs.unmarshalType); err != nil {
		return err
	}

	if len(cs.masterConfRaw) == 0 {
		return MasterProfileConfNotSetError
	}

	if err := fileUnmarshal(cs.masterConfRaw, v, cs.unmarshalType); err != nil {
		return err
	}

	return mergo.Merge(v, v1, mergo.WithOverride)
}

func loadConf(confDir ReadFileDir, cs *confStruct) error {
	dir, err := confDir.ReadDir(".")
	if err != nil {
		return err
	}
	for _, entry := range dir {
		name := entry.Name()
		if !strings.Contains(name, cs.confPrefix) {
			continue
		}
		env := getEnv(name)
		if env != cs.profileActive && env != cs.confPrefix {
			continue
		}
		fileRaw, err := confDir.ReadFile(name)
		if err != nil {
			return err
		}
		if env == cs.profileActive {
			cs.activeConfRaw = fileRaw
		} else {
			cs.masterConfRaw = fileRaw
		}
	}
	return nil
}

// getEnv 获取conf-***.yaml配置文件的作用域,
// eg: conf-dev.yaml -> dev
// conf-dev-i18n.yaml -> dev-i18n
// conf.json -> conf
func getEnv(filename string) string {
	return filename[strings.Index(filename, "-")+1 : strings.LastIndex(filename, ".")]
}

func getActiveProfile(envName, defaultValue string) string {
	profile := os.Getenv(envName)
	if profile == "" {
		profile = defaultValue
	}
	return profile
}

func fileUnmarshal(fileRaw []byte, v interface{}, unmarshalType UnmarshalType) error {
	switch unmarshalType {
	case Yaml:
		return yaml.Unmarshal(fileRaw, v)
	case Json:
		return json.Unmarshal(fileRaw, v)
	}
	return nil
}
