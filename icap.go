package main

import (
	"errors"
	"github.com/google/uuid"
	"github.com/hazcod/icap"
	"log"
	"net/http"
)

const (
	headerAuthName = "x-company-auth"
	headerResearcherName = "x-company-researcher"
	headerTestName = "x-company-test"
	headerRequestName = "x-company-request"
)

func main() {
	if err := icap.ListenAndServe(":1344", icap.HandlerFunc(handleRequest)); err != nil {
		log.Fatal(err)
	}
}

func authenticateRequest(r *http.Request) error {
	if r == nil {
		return errors.New("invalid request")
	}

	token := r.Header.Get(headerAuthName)

	if token == "" {
		return errors.New("missing token")
	}

	if token != "foo" {
		return errors.New("invalid token")
	}

	// remove auth header for origin
	r.Header.Del(headerAuthName)

	return nil
}

func enrichRequest(r *http.Request) error {
	if r == nil {
		return errors.New("invalid request")
	}

	r.Header.Set(headerResearcherName, "hazcod")
	r.Header.Set(headerTestName, "00-00-00-00")
	r.Header.Set(headerRequestName, uuid.New().String())
	return nil
}

func enrichResponse(r *icap.Request) error {
	if r == nil {
		return errors.New("invalid icap request")
	}

	if r.Request == nil {
		return errors.New("no request found")
	}

	if r.Response == nil {
		return errors.New("no response found")
	}

	// to be sure if the origin server is reflective
	r.Response.Header.Del(headerAuthName)
	r.Response.Header.Del(headerTestName)
	r.Response.Header.Del(headerResearcherName)

	requestID := r.Request.Header.Get(headerRequestName)
	if requestID == "" {
		return errors.New("no requestid found in original request")
	}

	r.Response.Header.Set(headerRequestName, requestID)
	return nil
}

func handleRequest(w icap.ResponseWriter, req *icap.Request) {
	switch req.Method {
	case "OPTIONS":
		log.Println("OPTIONS")

		// dirty fix to use only one http handler
		supportedMethod := "REQMOD"
		if req.URL.Path == "/response" {
			supportedMethod = "REQRESP"
		}

		h := w.Header()
		h.Set("Methods", supportedMethod)
		h.Set("Allow", "200,204")
		h.Set("Preview", "0")
		h.Set("Transfer-Preview", "*")
		w.WriteHeader(http.StatusOK, nil, false)
		return

	case "REQMOD":
		log.Println("modifying request: " + req.Request.Method + " " + req.Request.URL.String())

		if req.Request.Method == http.MethodConnect {
			log.Println("skipping CONNECT")
			w.WriteHeader(http.StatusNoContent, nil, false)
			return
		}

		// authenticate request
		if err := authenticateRequest(req.Request); err != nil {
			log.Printf("unauthenticated request: %+v", err)
			w.WriteHeader(http.StatusBadRequest, nil, false)
			return
		}

		// enrich request
		if err := enrichRequest(req.Request); err != nil {
			log.Printf("failed to enrich request: %+v", err)
			w.WriteHeader(http.StatusBadRequest, nil, false)
			return
		}

		// now return modified request
		log.Printf("returning enriched request")
		w.WriteHeader(http.StatusOK, req.Request, false)
		return

	case "RESPMOD":
		log.Println("modifying response: " + req.Request.Method + " " + req.Request.URL.String())

		// note: do not use req.Request.Response)
		if err := enrichResponse(req); err != nil {
			log.Printf("could not enrich response: %+v", err)
			w.WriteHeader(http.StatusBadRequest, nil, false)
			return
		}

		// now return modified request
		log.Printf("returning enriched response")
		w.WriteHeader(http.StatusOK, req.Response, false)
		return

	default:
		log.Println("Invalid request method: " + req.Method)
		w.WriteHeader(http.StatusMethodNotAllowed, nil, false)
		return
	}
}
