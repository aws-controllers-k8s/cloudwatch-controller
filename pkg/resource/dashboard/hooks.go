package dashboard

import (
	"encoding/json"
	"reflect"
)

func compareDashboardBody(a, b string) bool {
	equal, err := compareJSONObjects(a, b)

	// If there was an issue unmarshalling to JSON fallback to string comparison.
	if err != nil {
		return a == b
	}
	return equal
}

func compareJSONObjects(json1, json2 string) (bool, error) {
	var obj1, obj2 map[string]interface{}

	if err := json.Unmarshal([]byte(json1), &obj1); err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(json2), &obj2); err != nil {
		return false, err
	}

	return reflect.DeepEqual(obj1, obj2), nil
}
