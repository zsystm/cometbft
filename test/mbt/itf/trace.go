package itf

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// ITF spec: https://apalache.informal.systems/docs/adr/015adr-trace.html#the-itf-format
type Trace struct {
	Meta   *meta            `json:"#meta,omitempty"`
	Params map[string]*Expr `json:"params"`
	Vars   []string         `json:"vars"`
	States []*State         `json:"states"`
	Loop   int              `json:"loop,omitempty"`
}

type meta struct {
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
}

type State struct {
	Meta      map[string]any   `json:"#meta,omitempty"`
	VarValues map[string]*Expr `json:",omitempty"`
	// Index  int
}

func (s *State) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		return nil
	}
	var objmap map[string]json.RawMessage
	if err := json.Unmarshal(data, &objmap); err != nil {
		return err
	}

	var meta map[string]any
	values := make(map[string]*Expr)
	for k, v := range objmap {
		// fmt.Printf("unmarshalling \"%s\": %s\n", k, v)
		if k == "#meta" {
			if err := json.Unmarshal(v, &meta); err != nil {
				return err
			}
		} else {
			var expr Expr
			if err := json.Unmarshal(v, &expr); err != nil {
				return err
			}
			values[k] = &expr
		}
	}

	*s = State{Meta: meta, VarValues: values}

	return nil
}

type Expr struct {
	Value any
}

type (
	ListExprType = []Expr
	// MapExprType  = map[Expr]Expr
	MapExprType = map[string]Expr
)

func toExpr(parsed any) (Expr, error) {
	// fmt.Printf("toExpr: %+v\n", parsed)
	switch val := parsed.(type) {

	case nil:
		return Expr{}, fmt.Errorf("null value not allowed")

	case bool, string, float64:
		return Expr{val}, nil

	case []any:
		var list ListExprType
		for _, v := range val {
			if exp, err := toExpr(v); err != nil {
				return Expr{}, err
			} else {
				list = append(list, exp)
			}
		}
		return Expr{list}, nil

	case map[string]any:
		mapContent, is_map := val["#map"].([]any)
		if len(val) == 1 && is_map {
			map_ := make(MapExprType)
			for _, p := range mapContent {
				// fmt.Printf("key/value: %+v\n", p)
				pair, ok := p.([]any)
				if !ok {
					return Expr{}, fmt.Errorf("map entry is not a key/value pair: %v", p)
				}
				// fmt.Printf("key:'%s' value:%s\n", pair[0], pair[1])

				// if keyExpr, err := toExpr(pair[0]); err != nil {
				// 	return Expr{}, err
				if valExpr, err := toExpr(pair[1]); err != nil {
					return Expr{}, err
				} else {
					key := hashAny(pair[0])
					map_[key] = valExpr
				}
			}
			return Expr{map_}, nil
		}

		mapContent, is_tuple := val["#tup"].([]any)
		if len(val) == 1 && is_tuple {
			var elements ListExprType
			for _, elem := range mapContent {
				if exp, err := toExpr(elem); err != nil {
					return Expr{}, err
				} else {
					elements = append(elements, exp)
				}
			}
			return Expr{elements}, nil
		}

		mapContent, is_set := val["#set"].([]any)
		if len(val) == 1 && is_set {
			var elements ListExprType
			for _, elem := range mapContent {
				if exp, err := toExpr(elem); err != nil {
					return Expr{}, err
				} else {
					elements = append(elements, exp)
				}
			}
			return Expr{elements}, nil
		}

		// Otherwise, it's a record.
		// "Field names should not start with # and hence should not pose any collision with other constructs."
		record := make(MapExprType)
		for key, val := range val {
			if valExpr, err := toExpr(val); err != nil {
				return Expr{}, err
			} else {
				record[hashAny(key)] = valExpr
			}
		}
		return Expr{record}, nil

	// case map[any]any:
	// TODO

	default:
		return Expr{}, fmt.Errorf("type %T unexpected", parsed)
	}
}

func hashAny(x any) string {
	switch x := x.(type) {
	case map[string]string:
		hash := ""
		keys := make([]string, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			hash += k + x[k]
		}
		return hash
	case map[string]any:
		hash := ""
		keys := make([]string, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			hash += k + hashAny(x[k])
		}
		return hash
	case string:
		return x
	case nil:
		return ""
	default:
		return x.(string)
	}
}

func (e *Expr) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || string(data) == `""` {
		return nil
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	if expr, err := toExpr(parsed); err != nil {
		return err
	} else {
		*e = expr
	}

	return nil
}

func (trace *Trace) LoadFromFile(filePath string) error {
	// If file doesn't exist, do nothing.
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return err
	}

	// Load file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Unmarshal content
	if err := json.NewDecoder(file).Decode(trace); err != nil {
		return err
	}

	return nil
}
