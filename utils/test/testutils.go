package testutils

import (
	"reflect"
	"encoding/json"
	"fmt"
)

func AreEqualJSON(s1, s2 string) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal([]byte(s1), &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal([]byte(s2), &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}

func AreEqual(o1, o2 interface{}) (bool, error) {
	json1, _ := json.Marshal(o1)
	json2, _ := json.Marshal(o2)

	if _, err := AreEqualJSON(string(json1), string(json2)); err != nil {
		return false, err
	}

	return true, nil
}
