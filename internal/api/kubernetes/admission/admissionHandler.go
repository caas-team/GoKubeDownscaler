package admission

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type admissionHandler interface {
	// HandleValidation validates the admission request and returns an AdmissionResponse
	HandleValidation() *admissionv1.AdmissionResponse
}

// parseAdmissionReviewFromRequest extracts an AdmissionReview from a http.Request if possible.
func parseAdmissionReviewFromRequest(request *http.Request) (*admissionv1.AdmissionReview, error) {
	if request.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("Content-Type: %q should be %q",
			request.Header.Get("Content-Type"), "application/json")
	}

	bodybuf := new(bytes.Buffer)

	_, err := bodybuf.ReadFrom(request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	body := bodybuf.Bytes()

	if len(body) == 0 {
		return nil, errors.New("admission request body is empty")
	}

	var admissionReview admissionv1.AdmissionReview

	if err := json.Unmarshal(body, &admissionReview); err != nil {
		return nil, fmt.Errorf("could not parse admission review request: %w", err)
	}

	if admissionReview.Request == nil {
		return nil, errors.New("admission review can't be used: Request field is nil")
	}

	return &admissionReview, nil
}

// reviewResponse returns an AdmissionReview with the specified UID, allowed, httpCode, and reason.
func reviewResponse(uid types.UID, allowed bool, httpCode int32, reason string) *admissionv1.AdmissionReview {
	return &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionv1.AdmissionResponse{
			UID:     uid,
			Allowed: allowed,
			Result: &metav1.Status{
				Code:    httpCode,
				Message: reason,
			},
		},
	}
}

// sendAdmissionResponse sends the admission response to the client.
func sendAdmissionReviewResponse(writer http.ResponseWriter, output *admissionv1.AdmissionReview) {
	writer.Header().Set("Content-Type", "application/json")

	jout, err := json.Marshal(output)
	if err != nil {
		e := fmt.Sprintf("could not parse admission response: %v", err)
		slog.Error(e, slog.String("error", err.Error()))
		http.Error(writer, e, http.StatusInternalServerError)

		return
	}

	slog.Debug("sending response", "response", string(jout))

	_, writeErr := writer.Write(jout)
	if writeErr != nil {
		slog.Error("failed to write response", slog.String("error", writeErr.Error()))
	}
}
