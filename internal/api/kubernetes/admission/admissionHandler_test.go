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
		currentTest := testCase

		t.Run(currentTest.name, func(t *testing.T) {
			t.Parallel()

			req := currentTest.reqBuilder()
			parsed, err := parseAdmissionReviewFromRequest(req)

			if currentTest.expectError {
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

	response := newReviewResponse(uid, true, 200, "ok", false, false)
	if !response.Response.Allowed || response.Response.UID != uid {
		t.Errorf("unexpected response: %+v", response)
	}

	// dry-run + denied
	response = newReviewResponse(uid, false, 400, "denied", false, true)
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
	resp := newReviewResponse("id123", true, 200, "ok", false, false)

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

//nolint:paralleltest // cannot run test in parallel
func TestSendAdmissionReviewResponse_WriteError(t *testing.T) {
	// this test cannot run in parallel: test captures the global slog logger via slog.SetDefault,
	// which is shared across all goroutines. Running in parallel would cause a data race between
	// this test reading logBuffer and other parallel tests writing to the same global logger.
	// The correct long-term fix would involve injection a *slog.Logger into sendAdmissionReviewResponse.
	// This test will likely be refactored in the near future
	originalLogger := slog.Default()

	t.Cleanup(func() { slog.SetDefault(originalLogger) })

	var logBuffer bytes.Buffer

	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuffer, nil)))

	resp := newReviewResponse("id456", true, 200, "ok", false, false)
	sendAdmissionReviewResponse(&errorWriter{}, resp)

	capturedLogs := logBuffer.String()
	if !strings.Contains(capturedLogs, "failed to write response") {
		t.Errorf("expected log to contain 'failed to write response', got %q", capturedLogs)
	}

	if !strings.Contains(capturedLogs, "write error") {
		t.Errorf("expected log to contain 'write error', got %q", capturedLogs)
	}
}
