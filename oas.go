package main

import (
	"fmt"
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

	// Tags
	// ExternalDocs
}

type ServerObject struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`

	// Variables
}

type InfoObject struct {
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	TermsOfService string         `json:"termsOfService,omitempty"`
	Contact        *ContactObject `json:"contact,omitempty"`
	License        *LicenseObject `json:"license,omitempty"`
	Version        string         `json:"version"`
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
	Schema SchemaObject `json:"schema,omitempty"`
	// Example string       `json:"example,omitempty"`

	// Examples
	// Encoding
}

type SchemaObject struct {
	ID                 string              `json:"-"` // For goas
	PkgName            string              `json:"-"` // For goas
	FieldName          string              `json:"-"` // For goas
	DisabledFieldNames map[string]struct{} `json:"-"` // For goas

	Type        string                 `json:"type,omitempty"`
	Format      string                 `json:"format,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Properties  *orderedmap.OrderedMap `json:"properties,omitempty"`
	Description string                 `json:"description,omitempty"`
	Items       *SchemaObject          `json:"items,omitempty"` // use ptr to prevent recursive error
	Example     interface{}            `json:"example,omitempty"`
	Deprecated  bool                   `json:"deprecated,omitempty"`

	// Ref is used when SchemaObject is as a ReferenceObject
	Ref string `json:"$ref,omitempty"`

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
	// AllOf
	// OneOf
	// AnyOf
	// Not
	// AdditionalProperties
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

type ResponseObjectPackage struct {
	pkgPath      string
	pkgName      string
	jsonType     string
	goType       string
	desc         string
	p            *parser
	RO           *ResponseObject
	handlerError error
}

func newMediaTypeObjectCustomSchema(schema SchemaObject) *MediaTypeObject {
	return &MediaTypeObject{
		Schema: schema,
	}
}

func newMediaTypeObjectCustomRef(typeName string) *MediaTypeObject {
	return &MediaTypeObject{
		Schema: SchemaObject{
			Ref: addSchemaRefLinkPrefix(typeName),
		},
	}
}

func newMediaTypeObjectCustomType(mediaType string) *MediaTypeObject {
	return &MediaTypeObject{
		Schema: SchemaObject{
			Type: mediaType,
		},
	}
}

type ResponseObjectHandler interface {
	execute(*ResponseObjectPackage)
	setNext(ResponseObjectHandler)
}

type EmptyType struct {
	next ResponseObjectHandler
}

type TerminalMediaType struct {
	next ResponseObjectHandler
}
type RefType struct {
	next ResponseObjectHandler
}

type ComplexGoType struct {
	next ResponseObjectHandler
}

type SimpleGoType struct {
	next ResponseObjectHandler
}

type FailType struct {
	next ResponseObjectHandler
}

func (mt *EmptyType) execute(ROP *ResponseObjectPackage) {
	if ROP.jsonType != "" {
		mt.next.execute(ROP)
	}
}

func (mt *EmptyType) setNext(ROH ResponseObjectHandler) {
	mt.next = ROH
}

func (mt *FailType) execute(ROP *ResponseObjectPackage) {
	ROP.handlerError = fmt.Errorf("getResponseObject - unknown json type %v", ROP.jsonType)
}

func (mt *FailType) setNext(ROH ResponseObjectHandler) {
	mt.next = ROH
}

func (mt *TerminalMediaType) execute(ROP *ResponseObjectPackage) {
	if isTerminal(ROP.jsonType) {
		ROP.RO.Content[ContentTypeJson] = newMediaTypeObjectCustomType(cleanBrackets(ROP.jsonType))
		return
	}
	mt.next.execute(ROP)
}

func (mt *TerminalMediaType) setNext(ROH ResponseObjectHandler) {
	mt.next = ROH
}

func (mt *ComplexGoType) execute(ROP *ResponseObjectPackage) {
	if isComplex(ROP.jsonType) && isComplexGoType(ROP.goType) {
		schema, err := ROP.p.parseSchemaObject(ROP.pkgPath, ROP.pkgName, ROP.goType)
		if err != nil {
			ROP.p.debug("parseResponseComment: cannot parse goType", ROP.goType)
		}
		ROP.RO.Content[ContentTypeJson] = newMediaTypeObjectCustomSchema(*schema)
		return
	}
	mt.next.execute(ROP)

}
func (mt *ComplexGoType) setNext(ROH ResponseObjectHandler) {
	mt.next = ROH
}

func (mt *SimpleGoType) execute(ROP *ResponseObjectPackage) {
	typeName, err := ROP.p.registerType(ROP.pkgPath, ROP.pkgName, ROP.goType)
	if err != nil {
		ROP.handlerError = err
		return
	}

	if isComplex(ROP.jsonType) && isBasicGoType(typeName) {
		ROP.RO.Content[ContentTypeText] = newMediaTypeObjectCustomType("string")
		return
	}

	mt.next.execute(ROP)

}

func (mt *SimpleGoType) setNext(ROH ResponseObjectHandler) {
	mt.next = ROH
}

func (mt *RefType) execute(ROP *ResponseObjectPackage) {
	typeName, err := ROP.p.registerType(ROP.pkgPath, ROP.pkgName, ROP.goType)
	if err != nil {
		ROP.handlerError = err
		return
	}

	if isComplex(ROP.jsonType) && !isBasicGoType(typeName) {
		ROP.RO.Content[ContentTypeJson] = newMediaTypeObjectCustomRef(addSchemaRefLinkPrefix(typeName))
		return
	}

	mt.next.execute(ROP)

}

func (mt *RefType) setNext(ROH ResponseObjectHandler) {
	mt.next = ROH
}

func getResponseObject(ROP ResponseObjectPackage) (*ResponseObject, error) {
	responseObject := &ResponseObject{
		Content: map[string]*MediaTypeObject{},
	}
	responseObject.Description = strings.Trim(ROP.desc, "\"")

	ROP.RO = responseObject

	//linked list of ResponseObjects handlers

	emptyResponseObject := &EmptyType{}
	terminalResponseObject := &TerminalMediaType{}
	ComplexGoResponseObject := &ComplexGoType{}
	SimpleGoResponseObject := &SimpleGoType{}
	RefGoResponseObject := &RefType{}
	fail := &FailType{}

	emptyResponseObject.setNext(terminalResponseObject)
	terminalResponseObject.setNext(ComplexGoResponseObject)
	ComplexGoResponseObject.setNext(SimpleGoResponseObject)
	SimpleGoResponseObject.setNext(RefGoResponseObject)
	RefGoResponseObject.setNext(fail)

	emptyResponseObject.execute(&ROP)

	return ROP.RO, ROP.handlerError
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
