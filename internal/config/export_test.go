package config

func GetEnvAsBool(key string, defaultValue bool) bool {
	return getEnvAsBool(key, defaultValue)
}

func AllNonEmpty(keyValues map[string]string) error {
	return allNonEmpty(keyValues)
}

func AllNumbers(keyValues map[string]string) error {
	return allNumbers(keyValues)
}
