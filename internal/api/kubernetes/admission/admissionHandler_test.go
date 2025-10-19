package admission

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/types"
)

var errWriteError = errors.New("write error") // static error to satisfy err113

func TestParseAdmissionReviewFromRequest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		reqBuilder  func() *http.Request
		expectError bool
	}{
		{
			name: "valid request",
			reqBuilder: func() *http.Request {
				obj := &admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{UID: "1234"},
				}

				body, err := json.Marshal(obj)
				if err != nil {
					t.Fatalf("failed to marshal admission review: %v", err)
				}

				req := httptest.NewRequest(http.MethodPost, "/validate-workloads", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				return req
			},
			expectError: false,
		},
		{
			name: "wrong content type",
			reqBuilder: func() *http.Request {
				obj := &admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{UID: "1234"},
				}

				body, err := json.Marshal(obj)
				if err != nil {
					t.Fatalf("failed to marshal admission review: %v", err)
				}

				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
				req.Header.Set("Content-Type", "text/plain")

				return req
			},
			expectError: true,
		},
		{
			name: "empty body",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(nil))
				req.Header.Set("Content-Type", "application/json")

				return req
			},
			expectError: true,
		},
		{
			name: "invalid JSON",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("{bad json")))
				req.Header.Set("Content-Type", "application/json")

				return req
			},
			expectError: true,
		},
		{
			name: "missing Request field",
			reqBuilder: func() *http.Request {
				obj := &admissionv1.AdmissionReview{}

				body, err := json.Marshal(obj)
				if err != nil {
					t.Fatalf("failed to marshal admission review: %v", err)
				}

				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				return req
			},
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			req := testCase.reqBuilder()
			parsed, err := parseAdmissionReviewFromRequest(req)

			if testCase.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if parsed.Request == nil || parsed.Request.UID != "1234" {
					t.Fatalf("unexpected parsed request: %+v", parsed)
				}
			}
		})
	}
}

func TestReviewResponse(t *testing.T) {
	t.Parallel()

	uid := types.UID("abcd")

	response := newReviewResponse(uid, true, 200, "ok", false)
	if !response.Response.Allowed || response.Response.UID != uid {
		t.Errorf("unexpected response: %+v", response)
	}

	// dry-run + denied
	response = newReviewResponse(uid, false, 400, "denied", true)
	if !response.Response.Allowed {
		t.Errorf("expected Allowed=true in dry-run mode")
	}

	if response.Response.Result.Message != "denied (dry-run mode)" {
		t.Errorf("unexpected dry-run message: %s", response.Response.Result.Message)
	}
}

func TestPatchReviewResponse(t *testing.T) {
	t.Parallel()

	uid := types.UID("patchtest")
	patch := []byte(`[{"op":"add","path":"/metadata/labels/test","value":"true"}]`)

	response, err := newPatchReviewResponse(uid, patch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !response.Response.Allowed || !bytes.Equal(response.Response.Patch, patch) {
		t.Errorf("unexpected patch response: %+v", response)
	}
}

func TestSendAdmissionReviewResponse(t *testing.T) {
	t.Parallel()

	responseRecorder := httptest.NewRecorder()
	resp := newReviewResponse("id123", true, 200, "ok", false)

	sendAdmissionReviewResponse(responseRecorder, resp)

	if responseRecorder.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", responseRecorder.Code)
	}

	if got := responseRecorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", got)
	}
}

type errorWriter struct{}

func (e *errorWriter) Header() http.Header        { return http.Header{} }
func (e *errorWriter) WriteHeader(statusCode int) {}
func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, errWriteError
}

func TestSendAdmissionReviewResponse_WriteError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	resp := newReviewResponse("id456", true, 200, "ok", false)
	mockWriter := &errorWriter{}
	sendAdmissionReviewResponse(mockWriter, resp)

	logs := buf.String()
	if !strings.Contains(logs, "failed to write response") {
		t.Errorf("expected log to contain 'failed to write response', got %q", logs)
	}

	if !strings.Contains(logs, "write error") {
		t.Errorf("expected log to contain 'write error', got %q", logs)
	}
}
