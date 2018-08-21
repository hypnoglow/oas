package oas

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi"
)

func TestQueryValidator(t *testing.T) {
	handlers := OperationHandlers{
		"loginUser": http.HandlerFunc(handleUserLogin),
	}
	errHandler := makeErrorHandler()

	router := NewRouter(RouterMiddleware(QueryValidator(errHandler)))
	err := router.AddSpec(loadDocFile(t, "testdata/petstore_1.yml"), handlers)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	t.Run("positive", func(t *testing.T) {
		resp, _ := helperGet(t, router, "/v2/user/login?username=johndoe&password=123")
		expectedpayload := "username: johndoe, password: 123"
		if !bytes.Equal([]byte(expectedpayload), resp) {
			t.Fatalf("Expected response body to be\n%s\nbut got\n%s", expectedpayload, string(resp))
		}
	})

	t.Run("validation error", func(t *testing.T) {
		resp, _ := helperGet(t, router, "/v2/user/login?username=johndoe")
		expectedPayload := `{"errors":[{"message":"param password is required","field":"password"}]}`
		if !bytes.Equal([]byte(expectedPayload), resp) {
			t.Fatalf("Expected response body to be\n%s\nbut got\n%s", expectedPayload, string(resp))
		}
	})

	t.Run("request an url which handler does not provide operation context", func(t *testing.T) {
		resourceHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			fmt.Fprint(w, "hit no operation resource")
		})
		var panicmsg string
		handler := PanicRecover(QueryValidator(errHandler)(resourceHandler), &panicmsg)
		noopRouter := chi.NewRouter()
		noopRouter.Handle("/resource", handler)

		helperGet(t, noopRouter, "/resource")
		expectedPanic := "request has no OpenAPI parameters in its context"
		if panicmsg != expectedPanic {
			t.Fatalf("Expected panic %q but got %q", expectedPanic, panicmsg)
		}
	})
}

func handleUserLogin(w http.ResponseWriter, req *http.Request) {
	username := req.URL.Query().Get("username")
	password := req.URL.Query().Get("password")

	// Never do this! This is just for testing purposes.
	fmt.Fprintf(w, "username: %s, password: %s", username, password)
}
