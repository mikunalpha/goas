package main

const (
	OpenAPIVersion = "3.0.0"

	ContentTypeJson = "application/json"
)

type OASSpecObject struct {
	OpenAPI    string                `json:"openapi"`
	Info       *InfoObject           `json:"info"`
	Servers    []*ServerObject       `json:"servers"`
	Paths      PathsObject           `json:"paths"`
	Components *ComponentsOjbect     `json:"components,omitempty"`
	Security   []map[string][]string `json:"security,omitempty"`
}

type InfoObject struct {
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	TermsOfService string         `json:"termsOfService,omitempty"`
	Contact        *ContactObject `json:"contact,omitempty"`
	License        *LicenseObject `json:"license,omitempty"`
	Version        string         `json:"version"`
}

type ServerObject struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
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
	// Servers []ServerObject `json:"servers,omitempty"`
	Parameters []interface{} `json:"parameters,omitempty"`
}

type OperationObject struct {
	Tags        []string `json:"tags,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	// ExternalDocs *ExternalDocumentationObject `json:"externalDocs,omitempty"`
	OperationId string                     `json:"operationId,omitempty"`
	Parameters  []*ParameterObject         `json:"parameters,omitempty"`
	RequestBody *RequestBodyObject         `json:"requestBody,omitempty"`
	Responses   map[string]*ResponseObject `json:"responses,omitempty"`
	// Callbacks Map[string, Callback Object | Reference Object] `json:"callbacks,omitempty"`
	Deprecated bool `json:"deprecated,omitempty"`
	// Security   []*SecurityRequirementObject `json:"security,omitempty"`
	// Servers []ServerObject `json:"servers,omitempty"`

	// parser       *Parser
	packageName string
}

type ParameterObject struct {
	Name            string        `json:"name"`
	In              string        `json:"in"`
	Description     string        `json:"description,omitempty"`
	Required        bool          `json:"required,omitempty"`
	Deprecated      bool          `json:"deprecated,omitempty"`
	AllowEmptyValue bool          `json:"allowEmptyValue,omitempty"`
	Style           string        `json:"style,omitempty"`
	Explode         bool          `json:"explode,omitempty"`
	AllowReserved   bool          `json:"allowReserved,omitempty"`
	Schema          *SchemaObject `json:"schema,omitempty"`
	Example         string        `json:"example,omitempty"`
	// Examples map[string]ExampleObject `json:"examples,omitempty"`

	// Ref is for ReferenceObject
	Ref string `json:"$ref,omitempty"`
}

type SchemaObject struct {
	Type       string                 `json:"type,omitempty"`
	Format     string                 `json:"format,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Items      *ReferenceObject       `json:"items,omitempty"`
	Example    string                 `json:"example,omitempty"`

	// Ref is for ReferenceObject
	Ref string `json:"$ref,omitempty"`
}

type ReferenceObject struct {
	Ref string `json:"$ref,omitempty"`
}

type RequestBodyObject struct {
	Description string                      `json:"description,omitempty"`
	Content     map[string]*MediaTypeObject `json:"content"`
	Required    bool                        `json:"required,omitempty"`

	// Ref is for ReferenceObject
	Ref string `json:"$ref,omitempty"`
}

type MediaTypeObject struct {
	Schema  *SchemaObject `json:"schema,omitempty"`
	Example string        `json:"example,omitempty"`
	// Examples map[string]ExampleObject `json:"examples,omitempty"`
	Encoding map[string]*EncodingObject `json:"encoding,omitempty"`
}

type EncodingObject struct {
	ContentType   string                   `json:"contentType,omitempty"`
	Headers       map[string]*HeaderObject `json:"headers,omitempty"`
	Style         string                   `json:"style,omitempty"`
	Explode       bool                     `json:"explode,omitempty"`
	AllowReserved bool                     `json:"allowReserved,omitempty"`
}

type ResponseObject struct {
	Description string                      `json:"description"`
	Headers     map[string]*HeaderObject    `json:"headers,omitempty"`
	Content     map[string]*MediaTypeObject `json:"content"`

	// Ref is for ReferenceObject
	Ref string `json:"$ref,omitempty"`
}

type HeaderObject struct {
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

type ComponentsOjbect struct {
	SecuritySchemes map[string]*SecuritySchemeObject `json:"securitySchemes,omitempty"`
	Schemas         map[string]*SchemaObject         `json:"schemas,omitempty"`
	Responses       map[string]*ResponseObject       `json:"responses,omitempty"`
	Parameters      map[string]*ParameterObject      `json:"parameters,omitempty"`
}

type SecuritySchemeObject struct {
	Type string `json:"type"`
	Name string `json:"name"`
	In   string `json:"in"`
}
