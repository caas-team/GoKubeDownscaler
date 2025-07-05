package admission

type ContentTypeError struct {
	message     string
	contentType string
}

func newContentTypeError(message, contentType string) error {
	return &ContentTypeError{message: message, contentType: contentType}
}

func (a *ContentTypeError) Error() string {
	return a.message
}

type RequestBodyIsEmptyError struct {
	message string
}

func newRequestBodyIsEmptyError(message string) error {
	return &RequestBodyIsEmptyError{message: message}
}

func (a *RequestBodyIsEmptyError) Error() string {
	return a.message
}

type FailedToReadBodyError struct {
	message string
}

func newFailedToReadBodyError(message string) error {
	return &FailedToReadBodyError{message: message}
}

func (a *FailedToReadBodyError) Error() string {
	return a.message
}

type FailedToParseRequestError struct {
	message string
	error   error
}

func newFailedToParseRequestError(message string, err error) error {
	return &FailedToParseRequestError{message: message, error: err}
}

func (a *FailedToParseRequestError) Error() string {
	return a.message
}

type RequestFieldIsNilError struct {
	message string
}

func newRequestFieldIsNilError(message string) error {
	return &RequestFieldIsNilError{message: message}
}

func (a *RequestFieldIsNilError) Error() string {
	return a.message
}
