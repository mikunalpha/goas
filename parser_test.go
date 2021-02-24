package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	p, err := newParser("example/", "example/main.go", "", false)
	require.NoError(t, err)

	err = p.parse()
	require.NoError(t, err)

	bts, err := json.Marshal(p.OpenAPI)
	require.NoError(t, err)

	expected := `
{
   "openapi":"3.0.0",
   "info":{
      "title":"LaunchDarkly REST API",
      "description":"Build custom integrations with the LaunchDarkly REST API",
      "contact":{
         "name":"LaunchDarkly Technical Support Team",
         "url":"https://support.launchdarkly.com",
         "email":"support@launchdarkly.com"
      },
      "license":{
         "name":"Apache 2.0",
         "url":"https://www.apache.org/licenses/LICENSE-2.0"
      },
      "version":"2.0"
   },
   "servers":[
      {
         "url":"https://app.launchdarkly.com"
      }
   ],
   "paths":{
      
   },
   "components":{
      "securitySchemes":{
         "ApiKey":{
            "type":"apiKey",
            "in":"header",
            "name":"Authorization"
         }
      }
   },
   "security":[
      {
         "ApiKey":[
            "read",
            "write"
         ]
      }
   ]
}
`
	require.JSONEq(t, expected, string(bts))
}
