// SPDX-License-Identifier: PolyForm-Noncommercial-1.0.0

// Package fault defines stable machine-readable prototype error codes.
package fault

import (
	"errors"
	"fmt"
)

// Code is a stable error category suitable for CLI and API callers.
type Code string

const (
	// CodeInvalidInput indicates malformed or rejected caller input.
	CodeInvalidInput        Code = "INVALID_INPUT"
	CodeLimitExceeded       Code = "LIMIT_EXCEEDED"
	CodeCanceled            Code = "CANCELED"
	CodeRenderFailed        Code = "RENDER_FAILED"
	CodeWriteFailed         Code = "WRITE_FAILED"
	CodeOutputExists        Code = "OUTPUT_EXISTS"
	CodeSchemaMismatch      Code = "INPUT_SCHEMA_MISMATCH"
	CodeSchemaUnsupported   Code = "INPUT_SCHEMA_UNSUPPORTED"
	CodeDigestMismatch      Code = "INPUT_DIGEST_MISMATCH"
	CodeCorrelationMismatch Code = "INPUT_CORRELATION_MISMATCH"
)

// Error is the typed error returned across prototype package boundaries.
type Error struct {
	Code   Code
	Op     string
	Field  string
	Detail string
	Cause  error
}

// Error returns a stable, human-readable representation without a stack trace.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	message := string(e.Code)
	if e.Op != "" {
		message += " " + e.Op
	}
	if e.Field != "" {
		message += " [" + e.Field + "]"
	}
	if e.Detail != "" {
		message += ": " + e.Detail
	} else if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

// Unwrap exposes the underlying cause for errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.Cause }

// New constructs a typed error without an underlying cause.
func New(code Code, op, field, detail string) error {
	return &Error{Code: code, Op: op, Field: field, Detail: detail}
}

// Wrap constructs a typed error around cause.
func Wrap(code Code, op string, cause error) error {
	if cause == nil {
		return nil
	}
	return &Error{Code: code, Op: op, Cause: cause}
}

// WrapField constructs a typed error around cause and identifies its field.
func WrapField(code Code, op, field string, cause error) error {
	if cause == nil {
		return nil
	}
	return &Error{Code: code, Op: op, Field: field, Cause: cause}
}

// CodeOf returns a typed error's code or INVALID_INPUT for unknown errors.
func CodeOf(err error) Code {
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	return CodeInvalidInput
}

// ExitCode converts stable categories into CLI exit status values.
func ExitCode(err error) int {
	switch CodeOf(err) {
	case CodeInvalidInput, CodeLimitExceeded, CodeOutputExists, CodeSchemaMismatch, CodeSchemaUnsupported, CodeDigestMismatch, CodeCorrelationMismatch:
		return 2
	case CodeCanceled:
		return 130
	case CodeRenderFailed, CodeWriteFailed:
		return 1
	default:
		return 1
	}
}

// Format is a small helper for bounded validation details.
func Format(code Code, op, field, format string, args ...any) error {
	return New(code, op, field, fmt.Sprintf(format, args...))
}
