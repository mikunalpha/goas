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
      "/api/v2/foo":{
         "get":{
            "responses":{
               "200":{
                  "description":"Successful foo response",
                  "content":{
                     "application/json":{
                        "schema":{
                           "$ref":"#/components/schemas/FooResponse"
                        }
                     }
                  }
               },
               "401":{
                  "description":"Invalid access token"
               },
               "403":{
                  "description":"Forbidden"
               },
               "404":{
                  "description":"Invalid resource identifier"
               }
            },
            "summary":"Get all foos",
            "description":" Get all foos"
         },
         "put":{
            "responses":{
               "200":{
                  "description":"Successful foo response",
                  "content":{
                     "application/json":{
                        "schema":{
                           "$ref":"#/components/schemas/FooResponse"
                        }
                     }
                  }
               },
               "401":{
                  "description":"Invalid access token"
               },
               "403":{
                  "description":"Forbidden"
               },
               "404":{
                  "description":"Invalid resource identifier"
               }
            },
            "summary":"Put foo",
            "description":" Overwrite a foo"
         }
      },
      "/api/v2/foo/{id}/inner":{
         "put":{
            "responses":{
               "200":{
                  "description":"Successful innerfoo response",
                  "content":{
                     "application/json":{
                        "schema":{
                           "$ref":"#/components/schemas/InnerFoo"
                        }
                     }
                  }
               },
               "401":{
                  "description":"Invalid access token"
               },
               "403":{
                  "description":"Forbidden"
               },
               "404":{
                  "description":"Invalid resource identifier"
               }
            },
            "summary":"Get inner foos",
            "description":" Get Inner Foos"
         }
      }
   },
   "components":{
      "schemas":{
         "FooResponse":{
            "type":"object",
            "properties":{
               "id":{
                  "type":"string"
               },
               "bar":{
                  "type":"string"
               },
               "baz":{
                  "type":"string"
               },
               "startDate":{
                  "type":"string",
                  "format":"date-time"
               },
               "msg":{
                  "type":"object"
               },
               "foo":{
                  "type":"array",
                  "items":{
                     "type":"object",
                     "properties":{
                        "a":{
                           "type":"string"
                        },
                        "b":{
                           "type":"string"
                        }
                     }
                  }
               }
            }
         },
         "InnerFoo":{
            "type":"object",
            "properties":{
               "a":{
                  "type":"string"
               },
               "b":{
                  "type":"string"
               }
            }
         }
      },
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

func TestDeterministic(t *testing.T) {
	var allOutputs []string
	for i := 0; i < 100; i++ {
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
