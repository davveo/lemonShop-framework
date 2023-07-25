package config

type Option func(cs *confStruct)

type UnmarshalType string

const (
	Json = UnmarshalType("json")
	Yaml = UnmarshalType("yaml")
)

func FilePrefix(prefix string) Option {
	return func(cs *confStruct) {
		cs.confPrefix = prefix
	}
}

func ProfileFunc(f func() string) Option {
	return func(cs *confStruct) {
		cs.activeProfileFunc = f
	}
}

func ProfileUseEnv(envName, defaultProfile string) Option {
	return func(cs *confStruct) {
		cs.envName = envName
		cs.defaultEnvValue = defaultProfile
	}
}

func UnmarshalWith(uType UnmarshalType) Option {
	return func(cs *confStruct) {
		cs.unmarshalType = uType
	}
}
