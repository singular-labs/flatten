// Flatten makes flat, one-dimensional maps from arbitrarily nested ones.
//
// Map keys turn into compound
// names, like `a.b.1.c` (dotted style) or `a[b][1][c]` (Rails style).  It takes input as either JSON strings or
// Go structures.  It (only) knows how to traverse JSON types: maps, slices and scalars.
//
// You can flatten JSON strings.
//
//	nested := `{
//	  "one": {
//	    "two": [
//	      "2a",
//	      "2b"
//	    ]
//	  },
//	  "side": "value"
//	}`
//
//	flat, err := FlattenString(nested, "", DotStyle)
//
//	// output: `{ "one.two.0": "2a", "one.two.1": "2b", "side": "value" }`
//
// Or Go maps directly.
//
//	t := map[string]interface{}{
//		"a": "b",
//		"c": map[string]interface{}{
//			"d": "e",
//			"f": "g",
//		},
//		"z": 1.4567,
//	}
//
//	flat, err := Flatten(nested, "", RailsStyle)
//
//	// output:
//	// map[string]interface{}{
//	//	"a":    "b",
//	//	"c[d]": "e",
//	//	"c[f]": "g",
//	//	"z":    1.4567,
//	// }
//
package flatten

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
)

// The presentation style of keys.
type SeparatorStyle int

const (
	_ SeparatorStyle = iota

	// Separate nested key components with dots, e.g. "a.b.1.c.d"
	DotStyle

	// Separate nested key components with slashs, e.g. "a/b/1/c/d"
	SlashStyle

	// Separate ala Rails, e.g. "a[b][c][1][d]"
	RailsStyle
)

// Nested input must be a map or slice
var NotValidInputError = errors.New("Not a valid input: map or slice")

// Flatten generates a flat map from a nested one.  The original may include values of type map, slice and scalar,
// but not struct.  Keys in the flat map will be a compound of descending map keys and slice iterations.
// The presentation of keys is set by style.  A prefix is joined to each key.
func Flatten(nested map[string]interface{}, prefix string, style SeparatorStyle) (map[string]interface{}, error) {
	flatmap := make(map[string]interface{})

	err := flatten(true, flatmap, nested, prefix, style)
	if err != nil {
		return nil, err
	}

	return flatmap, nil
}

// FlattenAll generates a flat array from a nested map.  The original may include values of type map, slice
// and scalar, but not struct.  Items in the flat array will be a compound of descending map keys and slice
// iterations.  The presentation of keys is set by style.  A prefix is joined to each key.
func FlattenAll(nested interface{}, prefix string, style SeparatorStyle, sorted bool) ([]string, error) {
	result := []string{}

	err := flattenAll(true, &result, nested, prefix, style)
	if err != nil {
		return nil, err
	}

	if sorted {
		sort.Strings(result)
	}
	return result, nil
}

// FlattenString generates a flat JSON map from a nested one.  Keys in the flat map will be a compound of
// descending map keys and slice iterations.  The presentation of keys is set by style.  A prefix is joined
// to each key.
func FlattenString(nestedstr, prefix string, style SeparatorStyle) (string, error) {
	var nested map[string]interface{}
	err := json.Unmarshal([]byte(nestedstr), &nested)
	if err != nil {
		return "", err
	}

	flatmap, err := Flatten(nested, prefix, style)
	if err != nil {
		return "", err
	}

	flatb, err := json.Marshal(&flatmap)
	if err != nil {
		return "", err
	}

	return string(flatb), nil
}

func allPrimitives(arr []interface{}) bool {
	for _, item := range arr {
		switch item.(type) {
		case string, int32, int64, float32, float64, json.Number:
			continue
		default:
			return false
		}
	}
	return true
}

func flatten(top bool, flatMap map[string]interface{}, nested interface{}, prefix string, style SeparatorStyle) error {
	assign := func(newKey string, v interface{}) error {
		shouldFlatten := false

		switch v.(type) {
		case []interface{}:
			shouldFlatten = !allPrimitives(v.([]interface{}))
		case map[string]interface{}:
			shouldFlatten = true
		}

		if !shouldFlatten {
			flatMap[newKey] = v
			return nil
		}

		err := flatten(false, flatMap, v, newKey, style)
		return err
	}

	switch nested.(type) {
	case map[string]interface{}:
		for k, v := range nested.(map[string]interface{}) {
			newKey := enkey(top, prefix, k, style)
			assign(newKey, v)
		}
	case []interface{}:
		for i, v := range nested.([]interface{}) {
			newKey := enkey(top, prefix, strconv.Itoa(i), style)
			assign(newKey, v)
		}
	default:
		return NotValidInputError
	}

	return nil
}

func flattenAll(top bool, result *[]string, nested interface{}, prefix string, style SeparatorStyle) error {
	assign := func(newKey string, v interface{}) error {
		switch v.(type) {

		case map[string]interface{}, []interface{}:
			if err := flattenAll(false, result, v, newKey, style); err != nil {
				return err
			}

		default:
			newKey := fmt.Sprintf("%s.%v", newKey, v)
			*result = append(*result, newKey)
		}

		return nil
	}

	switch nested.(type) {
	case map[string]interface{}:
		for k, v := range nested.(map[string]interface{}) {
			newKey := enkey(top, prefix, k, style)
			assign(newKey, v)
		}
	case []interface{}:
		for i, v := range nested.([]interface{}) {
			newKey := enkey(top, prefix, strconv.Itoa(i), style)
			assign(newKey, v)
		}
	default:
		return NotValidInputError
	}

	return nil
}

func enkey(top bool, prefix, subkey string, style SeparatorStyle) string {
	key := prefix

	if top {
		key += subkey
	} else {
		switch style {
		case DotStyle:
			key += "." + subkey
		case SlashStyle:
			key += "/" + subkey
		case RailsStyle:
			key += "[" + subkey + "]"
		}
	}

	return key
}
