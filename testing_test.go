package oas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

func assertNoError(err error) {
	if err != nil {
		panic(err)
	}
}

func mustWriteBadRequest(w http.ResponseWriter, contentType string, b []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusBadRequest)
	_, err := w.Write(b)
	assertNoError(err)
}

// problemHandlerResponseWriter returns a ProblemHandlerFunc that writes to the
// response.
func problemHandlerResponseWriter() ProblemHandlerFunc {
	return func(problem Problem) {
		var b []byte
		var err error

		switch te := problem.Cause().(type) {
		case MultiError:
			b, err = json.Marshal(convertErrs(te.Errors()...))
		default:
			b, err = json.Marshal(convertErrs(te))
		}

		assertNoError(err)

		mustWriteBadRequest(
			problem.ResponseWriter(),
			"application/json",
			b,
		)
	}
}

// problemHandlerResponseWriter returns a ProblemHandlerFunc that logs to the
// buffer.
func problemHandlerBufferLogger(buf *bytes.Buffer) ProblemHandlerFunc {
	l := log.New(buf, "", 0)

	return func(problem Problem) {
		switch te := problem.Cause().(type) {
		case MultiError:
			for _, e := range convertErrs(te.Errors()...).Errors {
				l.Printf("problem handler: %s: %s", te.Message(), buildErrMessage(e))
			}
		default:
			l.Printf("problem handler: %s", te.Error())
		}
	}
}

// ---

type (
	errorItem struct {
		Message  string      `json:"message"`
		HasField bool        `json:"-"`
		Field    string      `json:"field,omitempty"`
		HasValue bool        `json:"-"`
		Value    interface{} `json:"value,omitempty"`
	}
	payload struct {
		Errors []errorItem `json:"errors"`
	}
)

func convertErrs(errs ...error) payload {
	// This is an example of composing an error for response from validation
	// errors.

	type fielder interface {
		Field() string
	}

	type valuer interface {
		Value() interface{}
	}

	p := payload{Errors: make([]errorItem, 0)}
	for _, e := range errs {
		item := errorItem{Message: e.Error()}
		if fe, ok := e.(fielder); ok {
			item.Field = fe.Field()
			item.HasField = true
		}
		if ve, ok := e.(valuer); ok {
			item.Value = ve.Value()
			item.HasValue = true
		}
		p.Errors = append(p.Errors, item)
	}

	return p
}

func buildErrMessage(err errorItem) string {
	msg := ""
	if err.HasField {
		msg += "field=" + err.Field + " "
	}
	if err.HasValue {
		msg += fmt.Sprintf("value=%v", err.Value) + " "
	}
	msg += "message=" + err.Message
	return msg
}

func loadDocFile(t *testing.T, fpath string) *Document {
	doc, err := LoadFile(fpath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	return doc
}

func loadDocBytes(b []byte) *Document {
	yml, err := swag.BytesToYAMLDoc(b)
	if err != nil {
		log.Fatalf("failed to convert spec to yaml: %v", err)
	}
	jsn, err := swag.YAMLToJSON(yml)
	if err != nil {
		log.Fatalf("failed to convert yaml to json: %v", err)
	}

	doc, err := loads.Analyzed(jsn, "2.0")
	if err != nil {
		log.Fatalf("failed to analyze spec: %v", err)
	}

	doc, err = doc.Expanded()
	if err != nil {
		log.Fatalf("failed to expand spec: %v", err)
	}

	if err := validate.Spec(doc, strfmt.Default); err != nil {
		log.Fatalf("failed to validate spec: %v", err)
	}

	return wrapDocument(doc)
}

var petstore = []byte(`
swagger: "2.0"
info:
  description: "This is a sample server Petstore server."
  version: "1.0.0"
  title: "Swagger Petstore"
  termsOfService: "http://swagger.io/terms/"
  contact:
    email: "apiteam@swagger.io"
  license:
    name: "Apache 2.0"
    url: "http://www.apache.org/licenses/LICENSE-2.0.html"
host: "petstore.swagger.io"
basePath: "/v2"
tags:
- name: "pet"
  description: "Everything about your Pets"
  externalDocs:
    description: "Find out more"
    url: "http://swagger.io"
schemes:
- "http"
paths:
  /pet:
    post:
      tags:
      - "pet"
      summary: "Add a new pet to the store"
      operationId: "addPet"
      consumes:
      - "application/json"
      produces:
      - "application/json"
      parameters:
      - in: "body"
        name: "body"
        description: "Pet object that needs to be added to the store"
        required: true
        schema:
          $ref: "#/definitions/Pet"
      - in: query
        name: debug
        type: boolean
      responses:
        405:
          description: "Invalid input"
      security:
      - petstore_auth:
        - "write:pets"
        - "read:pets"
  /pet/{petId}:
    get:
      tags:
      - "pet"
      summary: "Find pet by ID"
      description: "Returns a single pet"
      operationId: "getPetById"
      produces:
      - "application/json"
      parameters:
      - in: query
        name: debug
        type: boolean
      responses:
        200:
          description: "successful operation"
          schema:
            $ref: "#/definitions/Pet"
        400:
          description: "Invalid ID supplied"
        404:
          description: "Pet not found"
      security:
      - api_key: []
    parameters:
      - name: "petId"
        in: "path"
        description: "ID of pet to return"
        required: true
        type: "integer"
        format: "int64"
  /user/login:
    get:
      tags:
      - "user"
      summary: "Logs user into the system"
      description: ""
      operationId: "loginUser"
      produces:
      - "application/json"
      parameters:
      - name: "username"
        in: "query"
        description: "The user name for login"
        required: true
        type: "string"
      - name: "password"
        in: "query"
        description: "The password for login in clear text"
        required: true
        type: "string"
      responses:
        200:
          description: "successful operation"
          schema:
            type: "string"
          headers:
            X-Rate-Limit:
              type: "integer"
              format: "int32"
              description: "calls per hour allowed by the user"
            X-Expires-After:
              type: "string"
              format: "date-time"
              description: "date in UTC when token expires"
        400:
          description: "Invalid username/password supplied"
securityDefinitions:
  petstore_auth:
    type: "oauth2"
    authorizationUrl: "http://petstore.swagger.io/oauth/dialog"
    flow: "implicit"
    scopes:
      write:pets: "modify pets in your account"
      read:pets: "read your pets"
  api_key:
    type: "apiKey"
    name: "api_key"
    in: "header"
definitions:
  Pet:
    type: "object"
    required:
    - "name"
    - "age"
    properties:
      id:
        type: "integer"
        format: "int64"
      name:
        type: "string"
        example: "doggie"
      age:
        type: "integer"
        format: "int32"
        example: 7
      status:
        type: "string"
        description: "pet status in the store"
        enum:
        - "available"
        - "pending"
        - "sold"
  ApiResponse:
    type: "object"
    properties:
      code:
        type: "integer"
        format: "int32"
      type:
        type: "string"
      message:
        type: "string"
externalDocs:
  description: "Find out more about Swagger"
  url: "http://swagger.io"
`)
