package main

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/iancoleman/orderedmap"
)

const (
	OpenAPIVersion = "3.0.0"

	ContentTypeText = "text/plain"
	ContentTypeJson = "application/json"
	ContentTypeForm = "multipart/form-data"
)

type OpenAPIObject struct {
	OpenAPI string         `json:"openapi"` // Required
	Info    InfoObject     `json:"info"`    // Required
	Servers []ServerObject `json:"servers,omitempty"`
	Paths   PathsObject    `json:"paths"` // Required

	Components ComponentsOjbect      `json:"components,omitempty"` // Required for Authorization header
	Security   []map[string][]string `json:"security,omitempty"`

	Tags []TagDefinition `json:"tags,omitempty"`
	// ExternalDocs
}

type ServerObject struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`

	// Variables
}

type InfoObject struct {
	Title          string          `json:"title"`
	Description    *ReffableString `json:"description,omitempty"`
	TermsOfService string          `json:"termsOfService,omitempty"`
	Contact        *ContactObject  `json:"contact,omitempty"`
	License        *LicenseObject  `json:"license,omitempty"`
	Version        string          `json:"version"`
}

// Wrapper for a string that may be of the form `$ref:foo`
// denoting it should be json encoded as `{ "$ref": "foo" }`
type ReffableString struct {
	Value string
}

type reference struct {
	Ref string `json:"$ref"`
}

func (r ReffableString) MarshalJSON() ([]byte, error) {
	if strings.HasPrefix(r.Value, "$ref:") {
		if r.Value == "$ref:" {
			return nil, errors.New("$ref is missing URL")
		}
		// encode as a reference object instead of a string
		return json.Marshal(reference{Ref: r.Value[5:]})
	}
	return json.Marshal(r.Value)
}

type ContactObject struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

type LicenseObject struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type PathsObject map[string]*PathItemObject

type PathItemObject struct {
	Ref         string           `json:"$ref,omitempty"`
	Summary     string           `json:"summary,omitempty"`
	Description string           `json:"description,omitempty"`
	Get         *OperationObject `json:"get,omitempty"`
	Post        *OperationObject `json:"post,omitempty"`
	Patch       *OperationObject `json:"patch,omitempty"`
	Put         *OperationObject `json:"put,omitempty"`
	Delete      *OperationObject `json:"delete,omitempty"`
	Options     *OperationObject `json:"options,omitempty"`
	Head        *OperationObject `json:"head,omitempty"`
	Trace       *OperationObject `json:"trace,omitempty"`

	// Servers
	// Parameters
}

type OperationObject struct {
	Responses ResponsesObject `json:"responses"` // Required

	Tags        []string           `json:"tags,omitempty"`
	Summary     string             `json:"summary,omitempty"`
	Description string             `json:"description,omitempty"`
	Parameters  []ParameterObject  `json:"parameters,omitempty"`
	RequestBody *RequestBodyObject `json:"requestBody,omitempty"`
	OperationID string             `json:"operationId,omitempty"`
	// Tags
	// ExternalDocs
	// OperationID
	// Callbacks
	// Deprecated
	// Security
	// Servers
}

type ParameterObject struct {
	Name string `json:"name"` // Required
	In   string `json:"in"`   // Required. Possible values are "query", "header", "path" or "cookie"

	Description string        `json:"description,omitempty"`
	Required    bool          `json:"required,omitempty"`
	Example     interface{}   `json:"example,omitempty"`
	Schema      *SchemaObject `json:"schema,omitempty"`

	// Ref is used when ParameterOjbect is as a ReferenceObject
	Ref string `json:"$ref,omitempty"`

	// Deprecated
	// AllowEmptyValue
	// Style
	// Explode
	// AllowReserved
	// Examples
	// Content
}

type ReferenceObject struct {
	Ref string `json:"$ref,omitempty"`
}

type RequestBodyObject struct {
	Content map[string]*MediaTypeObject `json:"content"` // Required

	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`

	// Ref is used when RequestBodyObject is as a ReferenceObject
	Ref string `json:"$ref,omitempty"`
}

type MediaTypeObject struct {
	Schema  SchemaObject `json:"schema,omitempty"`
	Example interface{}  `json:"example,omitempty"`

	// Examples
	// Encoding
}

type SchemaObject struct {
	ID                 string              `json:"-"` // For goas
	PkgName            string              `json:"-"` // For goas
	FieldName          string              `json:"-"` // For goas
	DisabledFieldNames map[string]struct{} `json:"-"` // For goas

	Type                 *string                `json:"type,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Properties           *orderedmap.OrderedMap `json:"properties,omitempty"`
	AdditionalProperties *SchemaObject          `json:"additionalProperties,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Items                *SchemaObject          `json:"items,omitempty"` // use ptr to prevent recursive error
	Example              interface{}            `json:"example,omitempty"`
	Deprecated           bool                   `json:"deprecated,omitempty"`
	Enum                 []string               `json:"enum,omitempty"`

	// Ref is used when SchemaObject is as a ReferenceObject
	Ref string `json:"$ref,omitempty"`

	AllOf []*SchemaObject `json:"allOf,omitempty"`
	OneOf []*SchemaObject `json:"oneOf,omitempty"`
	AnyOf []*SchemaObject `json:"anyOf,omitempty"`
	Not   *SchemaObject   `json:"not,omitempty"`

	// Title
	// MultipleOf
	// Maximum
	// ExclusiveMaximum
	// Minimum
	// ExclusiveMinimum
	// MaxLength
	// MinLength
	// Pattern
	// MaxItems
	// MinItems
	// UniqueItems
	// MaxProperties
	// MinProperties
	// Enum
	// Description
	// Default
	// Nullable
	// ReadOnly
	// WriteOnly
	// XML
	// ExternalDocs
}

type ResponsesObject map[string]*ResponseObject // [status]ResponseObject

type ResponseObject struct {
	Description string `json:"description"` // Required

	Headers map[string]*HeaderObject    `json:"headers,omitempty"`
	Content map[string]*MediaTypeObject `json:"content,omitempty"`

	// Ref is for ReferenceObject
	Ref string `json:"$ref,omitempty"`

	// Links
}

type HeaderObject struct {
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`

	// Ref is used when HeaderObject is as a ReferenceObject
	Ref string `json:"$ref,omitempty"`
}

type ComponentsOjbect struct {
	Schemas         map[string]*SchemaObject         `json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecuritySchemeObject `json:"securitySchemes,omitempty"`

	// Responses
	// Parameters
	// Examples
	// RequestBodies
	// Headers
	// Links
	// Callbacks
}

type SecuritySchemeObject struct {
	// Generic fields
	Type        string `json:"type"` // Required
	Description string `json:"description,omitempty"`

	// http
	Scheme string `json:"scheme,omitempty"`

	// apiKey
	In   string `json:"in,omitempty"`
	Name string `json:"name,omitempty"`

	// OpenID
	OpenIdConnectUrl string `json:"openIdConnectUrl,omitempty"`

	// OAuth2
	OAuthFlows *SecuritySchemeOauthObject `json:"flows,omitempty"`

	// BearerFormat
}

type SecuritySchemeOauthObject struct {
	Implicit              *SecuritySchemeOauthFlowObject `json:"implicit,omitempty"`
	AuthorizationCode     *SecuritySchemeOauthFlowObject `json:"authorizationCode,omitempty"`
	ResourceOwnerPassword *SecuritySchemeOauthFlowObject `json:"password,omitempty"`
	ClientCredentials     *SecuritySchemeOauthFlowObject `json:"clientCredentials,omitempty"`
}

func (s *SecuritySchemeOauthObject) ApplyScopes(scopes map[string]string) {
	if s.Implicit != nil {
		s.Implicit.Scopes = scopes
	}

	if s.AuthorizationCode != nil {
		s.AuthorizationCode.Scopes = scopes
	}

	if s.ResourceOwnerPassword != nil {
		s.ResourceOwnerPassword.Scopes = scopes
	}

	if s.ClientCredentials != nil {
		s.ClientCredentials.Scopes = scopes
	}
}

type SecuritySchemeOauthFlowObject struct {
	AuthorizationUrl string            `json:"authorizationUrl,omitempty"`
	TokenUrl         string            `json:"tokenUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

type TagDefinition struct {
	Name        string          `json:"name"`
	Description *ReffableString `json:"description,omitempty"`
}
