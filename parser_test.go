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

	bts, err := json.MarshalIndent(p.OpenAPI, "", "    ")
	require.NoError(t, err)

	expected := `
{
    "components": {
        "schemas": {
            "Environment": {
                "properties": {
                    "name": {
                        "type": "string"
                    }
                },
                "type": "object"
            },
            "FooResponse": {
                "properties": {
					"count": {
                        "type": "integer"
                    },
                    "endDate": {
                        "type": "integer"
                    },
                    "environments": {
                        "additionalProperties": {
                            "properties": {
                                "name": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        },
                        "type": "object"
                    },
                    "foo": {
                        "items": {
                            "properties": {
                                "a": {
                                    "type": "string"
                                },
                                "b": {
                                    "type": "string"
                                }
                            },
                            "type": "object"
                        },
                        "type": "array"
                    },
                    "id": {
                        "type": "string"
                    },
                    "msg": {
                        "type": "object"
                    },
                    "startDate": {
                        "format": "date-time",
                        "type": "string"
                    }
                },
                "type": "object"
            },
            "InnerFoo": {
                "properties": {
                    "a": {
                        "type": "string"
                    },
                    "b": {
                        "type": "string"
                    }
                },
                "type": "object"
            }
        },
        "securitySchemes": {
            "ApiKey": {
                "in": "header",
                "name": "Authorization",
                "type": "apiKey"
            }
        }
    },
    "info": {
        "contact": {
            "email": "support@launchdarkly.com",
            "name": "LaunchDarkly Technical Support Team",
            "url": "https://support.launchdarkly.com"
        },
        "description": "Build custom integrations with the LaunchDarkly REST API",
        "license": {
            "name": "Apache 2.0",
            "url": "https://www.apache.org/licenses/LICENSE-2.0"
        },
        "title": "LaunchDarkly REST API",
        "version": "2.0"
    },
    "openapi": "3.0.0",
    "paths": {
        "/api/v2/foo": {
            "get": {
                "description": " Get all foos",
                "responses": {
                    "200": {
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/FooResponse"
                                }
                            }
                        },
                        "description": "Successful foo response"
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
                "summary": "Get all foos"
            },
            "put": {
                "description": " Overwrite a foo",
                "responses": {
                    "200": {
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/FooResponse"
                                }
                            }
                        },
                        "description": "Successful foo response"
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
                "summary": "Put foo"
            }
        },
        "/api/v2/foo/{id}/inner": {
            "put": {
                "description": " Get Inner Foos",
                "responses": {
                    "200": {
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/InnerFoo"
                                }
                            }
                        },
                        "description": "Successful innerfoo response"
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
                "summary": "Get inner foos"
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
    ],
    "servers": [
        {
            "url": "https://app.launchdarkly.com"
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
