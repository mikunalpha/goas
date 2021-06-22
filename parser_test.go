package main

import (
	"encoding/json"
	"fmt"
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

	expected := `
{
    "openapi": "3.0.0",
    "info": {
        "title": "LaunchDarkly REST API",
        "description": "Build custom integrations with the LaunchDarkly REST API",
        "contact": {
            "name": "LaunchDarkly Technical Support Team",
            "url": "https://support.launchdarkly.com",
            "email": "support@launchdarkly.com"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "https://www.apache.org/licenses/LICENSE-2.0"
        },
        "version": "2.0"
    },
    "servers": [
        {
            "url": "https://app.launchdarkly.com"
        }
    ],
    "paths": {
        "/api/v2/foo": {
            "get": {
                "responses": {
                    "200": {
                        "description": "Successful foo response",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/example.FooResponse"
                                }
                            }
                        }
                    },
                    "401": {
                        "description": "Invalid access token"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Invalid resource identifier"
                    }
                },
                "summary": "Get all foos",
                "description": " Get all foos",
                "operationId": "getAllFoos",
                "tags": [
                    "foo"
                ]
            },
            "put": {
                "responses": {
                    "200": {
                        "description": "Successful foo response",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/example.FooResponse"
                                }
                            }
                        }
                    },
                    "401": {
                        "description": "Invalid access token"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Invalid resource identifier"
                    }
                },
                "summary": "Put foo",
                "description": " Overwrite a foo"
            }
        },
        "/api/v2/foo/{id}/inner": {
            "put": {
                "responses": {
                    "200": {
                        "description": "Successful innerfoo response",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/example.InnerFoo"
                                }
                            }
                        }
                    },
                    "401": {
                        "description": "Invalid access token"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Invalid resource identifier"
                    }
                },
                "summary": "Get inner foos",
                "description": " Get Inner Foos"
            }
        },
        "/api/v2/vfoo": {
            "get": {
                "responses": {
                    "200": {
                        "description": "Successful foo response",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/example.FooResponse"
                                }
                            }
                        }
                    },
                    "401": {
                        "description": "Invalid access token"
                    },
                    "403": {
                        "description": "Forbidden"
                    },
                    "404": {
                        "description": "Invalid resource identifier"
                    }
                },
                "summary": "Get Foo as var",
                "description": " get a foo var",
                "tags": [
                    "vfoo"
                ]
            }
        }
    },
    "components": {
        "schemas": {
            "example.BsonID": {
                "type": "string"
            },
            "example.DoubleAlias": {
                "type": "object",
                "additionalProperties": {}
            },
            "example.Environment": {
                "type": "object",
                "properties": {
                    "name": {
                        "type": "string"
                    }
                }
            },
            "example.FooResponse": {
                "type": "object",
                "properties": {
                    "bsonId": {
                        "$ref": "#/components/schemas/example.BsonID"
                    },
                    "id": {
                        "type": "string"
                    },
                    "startDate": {
                        "type": "string",
                        "format": "date-time"
                    },
                    "endDate": {
                        "$ref": "#/components/schemas/example.UnixMillis"
                    },
                    "count": {
                        "type": "integer",
                        "format": "int64",
                        "example": 6
                    },
                    "msg": {
                        "type": "object"
                    },
                    "foo": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "a": {
                                    "type": "string"
                                },
                                "b": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "environments": {
                        "type": "object",
                        "additionalProperties": {
                            "type": "object",
                            "properties": {
                                "name": {
                                    "type": "string"
                                }
                            }
                        }
                    },
                    "freeForm": {},
                    "jsonMap": {
                        "$ref": "#/components/schemas/example.JsonMap"
                    },
                    "doubleAlias": {
                        "$ref": "#/components/schemas/example.DoubleAlias"
                    },
                    "interfaceBlah": {
                        "$ref": "#/components/schemas/example.InterfaceResponse"
                    },
                    "instruction": {
                        "$ref": "#/components/schemas/example.Instruction"
                    },
                    "bsonPtr": {
                        "example": "blah blah blah",
                        "$ref": "#/components/schemas/example.BsonID"
                    },
                    "randomBool": {
                        "example": true,
                        "type": "boolean"
                    }
                }
            },
            "example.InnerFoo": {
                "type": "object",
                "properties": {
                    "a": {
                        "type": "string"
                    },
                    "b": {
                        "type": "string"
                    }
                }
            },
            "example.Instruction": {
                "type": "object",
                "additionalProperties": {}
            },
            "example.InterfaceResponse": {},
            "example.JsonMap": {
                "type": "object",
                "additionalProperties": {}
            },
            "example.UnixMillis": {
                "type": "integer",
                "format": "int64"
            }
        },
        "securitySchemes": {
            "ApiKey": {
                "type": "apiKey",
                "in": "header",
                "name": "Authorization"
            }
        }
    },
    "security": [
        {
            "ApiKey": [
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
	require.Error(t, duplicateError, "Already exists, v2/foo/bar [get]")
}
