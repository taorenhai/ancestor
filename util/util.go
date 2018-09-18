package util

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/juju/errors"
)

// IsFileExists - whether the file exist
// @filename: file full path name
func IsFileExists(filename string) bool {
	fi, err := os.Stat(filename)
	if err != nil {
		return os.IsExist(err)
	}

	return !fi.IsDir()
}

// Min return the min number
func Min(a, b uint64) uint64 {
	if a > b {
		return b
	}
	return a
}

// Max return the max number
func Max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// Compare two byte variable
func Compare(a, b []byte) int {
	return bytes.Compare(a, b)
}

// SplitString split the string to some sub string array
func SplitString(s, step string) []string {
	if s == "" || step == "" {
		return nil
	}

	return strings.Split(s, step)
}

// ReadFile read the content of the file
func ReadFile(filename string) ([]byte, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Errorf("ReadFile file:%v error:%v", filename, err)
	}
	return bytes, nil
}

// DecodeJSON decode a bytes to Json object
func DecodeJSON(bytes []byte) (map[string]interface{}, error) {
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(bytes, &jsonMap); err != nil {
		return nil, errors.Errorf("DecodeJson error:%v", err.Error())
	}
	return jsonMap, nil
}

// EncodeJSON encode object to []byte
func EncodeJSON(data interface{}) ([]byte, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// CheckMapString check whether the map key's value type is string
func CheckMapString(m map[string]interface{}, key string) (string, error) {
	if _, ok := m[key]; !ok {
		return "", errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].(string); ok {
		return v, nil
	}
	return "", errors.Errorf(key + " is not string type")
}

// CheckMapBool check whether the map key's value type is bool
func CheckMapBool(m map[string]interface{}, key string) (bool, error) {
	if _, ok := m[key]; !ok {
		return false, errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].(bool); ok {
		return v, nil
	}
	return false, errors.Errorf(key + " is not bool type")
}

// CheckMapInt64 check whether the map key's value type is int64
func CheckMapInt64(m map[string]interface{}, key string) (int64, error) {
	if _, ok := m[key]; !ok {
		return 0, errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return int64(v), nil
	}
	return 0, errors.Errorf(key + " is not int64 type")
}

// CheckMapInt32 check whether the map key's value type is int32
func CheckMapInt32(m map[string]interface{}, key string) (int32, error) {
	if _, ok := m[key]; !ok {
		return 0, errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return int32(v), nil
	}

	return 0, errors.Errorf(key + " is not int32 type")
}

// CheckMapInt check whether the map key's value type is int
func CheckMapInt(m map[string]interface{}, key string) (int, error) {
	if _, ok := m[key]; !ok {
		return 0, errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return int(v), nil
	}
	return 0, errors.Errorf(key + " is not int type")
}

// CheckMapUInt16 check whether the map key's value type is int
func CheckMapUInt16(m map[string]interface{}, key string) (uint16, error) {
	if _, ok := m[key]; !ok {
		return 0, errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return uint16(v), nil
	}

	return 0, errors.Errorf(key + " is not uint16 type")
}

// CheckMapUInt32 check whether the map key's value type is uint32
func CheckMapUInt32(m map[string]interface{}, key string) (uint32, error) {
	if _, ok := m[key]; !ok {
		return 0, errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].(float64); ok {
		return uint32(v), nil
	}
	return 0, errors.Errorf(key + " is not uint32 type")
}

// CheckMapInterface check whether the map key's value type is interface{}
func CheckMapInterface(m map[string]interface{}, key string) (map[string]interface{}, error) {
	if _, ok := m[key]; !ok {
		return nil, errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].(interface{}); ok {
		return v.(map[string]interface{}), nil
	}
	return nil, errors.Errorf(key + " is not interface{} type")
}

// CheckMapInterfaceSlice check whether the map key's value type is []interface{}
func CheckMapInterfaceSlice(m map[string]interface{}, key string) ([]interface{}, error) {
	if _, ok := m[key]; !ok {
		return nil, errors.Errorf(key + " is nil")
	}

	if v, ok := m[key].([]interface{}); ok {
		return v, nil
	}
	return nil, errors.Errorf(key + " is not []interface{} type")
}

// CheckInterfaceMapInterface check whether the interface is map[string]interface{}
func CheckInterfaceMapInterface(i interface{}) (map[string]interface{}, error) {
	if v, ok := i.(interface{}); ok {
		return v.(map[string]interface{}), nil
	}
	return nil, errors.Errorf("%v is not interface{} type", i)
}
