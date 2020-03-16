package decrypt

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tnychn/gotube/errors"
)

type Decryption struct {
	transformPlan []string
	transformMap  map[string]func([]string, int) []string
}

func NewDecryption(js string) (*Decryption, error) {
	transformPlan, err := getTransformPlan(js)
	if err != nil {
		return nil, err
	}
	v := strings.Split(transformPlan[0], ".")[0]
	transformMap, err := getTransformMap(js, v)
	if err != nil {
		return nil, err
	}
	return &Decryption{transformPlan: transformPlan, transformMap: transformMap}, nil
}

func (d *Decryption) DecryptSignature(s string) (string, error) {
	sig := strings.Split(s, "")
	for _, jsFunc := range d.transformPlan {
		funcName, args, err := d.parseFunction(jsFunc)
		if err != nil {
			return "", err
		}
		sig = d.transformMap[funcName](sig, args)
	}
	signature := strings.Join(sig, "")
	return signature, nil
}

func (d *Decryption) parseFunction(jsFunc string) (string, int, error) {
	re := regexp.MustCompile(`\w+\.(\w+)\(\w,(\d+)\)`)
	parseMatch := re.FindStringSubmatch(jsFunc)
	if len(parseMatch) == 0 {
		return "", 0, errors.ExtractError{Caller: "parse function", Pattern: re.String()}
	}
	funcName := parseMatch[1]
	funcArg, err := strconv.Atoi(parseMatch[2])
	if err != nil {
		return "", 0, err
	}
	return funcName, funcArg, nil
}

func getTransformPlan(js string) ([]string, error) {
	getInitialFunctionName := func(js string) (string, error) {
		patterns := []string{
			`\b[cs]\s*&&\s*[adf]\.set\([^,]+\s*,\s*encodeURIComponent\s*\(\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
			`\b[a-zA-Z0-9]+\s*&&\s*[a-zA-Z0-9]+\.set\([^,]+\s*,\s*encodeURIComponent\s*\(\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
			`\b(?P<sig>[a-zA-Z0-9$]{2})\s*=\s*function\(\s*a\s*\)\s*{\s*a\s*=\s*a\.split\(\s*""\s*\)`,
			`(?P<sig>[a-zA-Z0-9$]+)\s*=\s*function\(\s*a\s*\)\s*{\s*a\s*=\s*a\.split\(\s*""\s*\)`,
			`(["\'])signature\1\s*,\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
			`\.sig\|\|(?P<sig>[a-zA-Z0-9$]+)\(`,
			`yt\.akamaized\.net/\)\s*\|\|\s*.*?\s*[cs]\s*&&\s*[adf]\.set\([^,]+\s*,\s*(?:encodeURIComponent\s*\()?\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
			`b[cs]\s*&&\s*[adf]\.set\([^,]+\s*,\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
			`\b[a-zA-Z0-9]+\s*&&\s*[a-zA-Z0-9]+\.set\([^,]+\s*,\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
			`\bc\s*&&\s*a\.set\([^,]+\s*,\s*\([^)]*\)\s*\(\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
			`\bc\s*&&\s*[a-zA-Z0-9]+\.set\([^,]+\s*,\s*\([^)]*\)\s*\(\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
			`\bc\s*&&\s*[a-zA-Z0-9]+\.set\([^,]+\s*,\s*\([^)]*\)\s*\(\s*(?P<sig>[a-zA-Z0-9$]+)\(`,
		}
		for _, pattern := range patterns {
			matches := regexp.MustCompile(pattern).FindStringSubmatch(js)
			if len(matches) > 0 {
				return matches[1], nil
			}
		}
		return "", errors.ExtractError{Caller: "initial function name", Pattern: "<initial function patterns>"}
	}
	ifunc, err := getInitialFunctionName(js)
	if err != nil {
		return nil, err
	}
	funcName := regexp.QuoteMeta(ifunc)
	pattern := fmt.Sprintf(`%v=function\(\w\){[a-z=\.\(\"\)]*;(.*);(?:.+)}`, funcName)
	matches := regexp.MustCompile(pattern).FindStringSubmatch(js)
	if len(matches) == 0 {
		return nil, errors.ExtractError{Caller: "transform plan", Pattern: pattern}
	}
	return strings.Split(matches[1], ";"), nil
}

func getTransformMap(js, v string) (map[string]func([]string, int) []string, error) {
	getTransformObject := func(js, v string) ([]string, error) {
		pattern := fmt.Sprintf(`(?s)var %v={(.*?)};`, regexp.QuoteMeta(v))
		re := regexp.MustCompile(pattern)
		transformMatch := re.FindStringSubmatch(js)
		if len(transformMatch) == 0 {
			return nil, errors.ExtractError{Caller: "transform object", Pattern: pattern}
		}
		return strings.Split(strings.ReplaceAll(transformMatch[1], "\n", " "), ", "), nil
	}

	transformObject, err := getTransformObject(js, v)
	if err != nil {
		return nil, err
	}
	mapper := make(map[string]func([]string, int) []string)
	for _, obj := range transformObject {
		splitted := strings.SplitN(obj, ":", 2)
		fn, err := mapFunctions(splitted[1])
		if err != nil {
			return nil, err
		}
		mapper[splitted[0]] = fn
	}
	return mapper, nil
}

func mapFunctions(jsFunc string) (func([]string, int) []string, error) {
	reverse := func(arr []string, _ int) []string {
		for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
			arr[i], arr[j] = arr[j], arr[i]
		}
		return arr
	}
	splice := func(arr []string, b int) []string {
		return append(arr[:b], arr[b*2:]...)
	}
	swap := func(arr []string, b int) []string {
		c := arr[0]
		arr[0] = arr[b%len(arr)]
		arr[b] = c
		return arr
	}
	var lastPattern string
	mapper := map[string]func([]string, int) []string{
		`{\w\.reverse\(\)}`:    reverse,
		`{\w\.splice\(0,\w\)}`: splice,
		`{var\s\w=\w\[0\];\w\[0\]=\w\[\w\%\w.length\];\w\[\w\]=\w}`:            swap,
		`{var\s\w=\w\[0\];\w\[0\]=\w\[\w\%\w.length\];\w\[\w\%\w.length\]=\w}`: swap,
	}
	for pattern, fn := range mapper {
		if matched, _ := regexp.MatchString(pattern, jsFunc); matched {
			return fn, nil
		}
		lastPattern = pattern
	}
	return nil, errors.ExtractError{Caller: "map functions", Pattern: lastPattern}
}
