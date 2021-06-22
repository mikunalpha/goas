package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	p, err := newParser("example/", "example/main.go", "", false)
	require.NoError(t, err)

	err = p.parse()
	require.NoError(t, err)

	bts, err := json.MarshalIndent(p.OpenAPI, "", "    ")
	require.NoError(t, err)

	fmt.Println(string(bts))

	expected, err := ioutil.ReadFile("./example/example.json")
	require.JSONEq(t, string(expected), string(bts))
}

func TestDeterministic(t *testing.T) {
	var allOutputs []string
	for i := 0; i < 10; i++ {
		p, err := newParser("example/", "example/main.go", "", false)
		require.NoError(t, err)

		err = p.parse()
		require.NoError(t, err)

		bts, err := json.Marshal(p.OpenAPI)
		require.NoError(t, err)
		allOutputs = append(allOutputs, string(bts))
	}

	for i := 0; i < len(allOutputs)-1; i++ {
		require.Equal(t, allOutputs[i], allOutputs[i+1])
	}
}

func Test_parseRouteComment(t *testing.T) {
	p, err := newParser("example/", "example/main.go", "", false)
	require.NoError(t, err)

	operation := &OperationObject{
		Responses: map[string]*ResponseObject{},
	}
	p.OpenAPI.Paths["v2/foo/bar"] = &PathItemObject{}
	p.OpenAPI.Paths["v2/foo/bar"].Get = operation

	duplicateError := p.parseRouteComment(operation, "@Router v2/foo/bar [get]")
	require.Error(t, duplicateError)
}
