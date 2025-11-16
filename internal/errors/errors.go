package errors

import (
	"fmt"
	"net/http"
)

type ErrorCode string

const (
	ErrCodeTeamExists   ErrorCode = "TEAM_EXISTS"
	ErrCodePRExists     ErrorCode = "PR_EXISTS"
	ErrCodePRMerged     ErrorCode = "PR_MERGED"
	ErrCodeNotAssigned  ErrorCode = "NOT_ASSIGNED"
	ErrCodeNoCandidate  ErrorCode = "NO_CANDIDATE"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrCodeBadRequest   ErrorCode = "BAD_REQUEST"
)

type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	HTTPStatus int       `json:"-"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type ErrorResponse struct {
	Error AppError `json:"error"`
}

func NewAppError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}


func ErrTeamExists(teamName string) *AppError {
	return NewAppError(
		ErrCodeTeamExists,
		fmt.Sprintf("team '%s' already exists", teamName),
		http.StatusBadRequest,
	)
}

func ErrPRExists(prID string) *AppError {
	return NewAppError(
		ErrCodePRExists,
		fmt.Sprintf("pull request '%s' already exists", prID),
		http.StatusConflict,
	)
}

func ErrPRMerged() *AppError {
	return NewAppError(
		ErrCodePRMerged,
		"cannot modify merged pull request",
		http.StatusConflict,
	)
}

func ErrNotAssigned() *AppError {
	return NewAppError(
		ErrCodeNotAssigned,
		"user is not assigned as reviewer",
		http.StatusConflict,
	)
}

func ErrNoCandidate() *AppError {
	return NewAppError(
		ErrCodeNoCandidate,
		"no active candidates available for assignment",
		http.StatusConflict,
	)
}

func ErrNotFound(resource string) *AppError {
	return NewAppError(
		ErrCodeNotFound,
		fmt.Sprintf("%s not found", resource),
		http.StatusNotFound,
	)
}

func ErrInternal(err error) *AppError {
	return NewAppError(
		ErrCodeInternal,
		fmt.Sprintf("internal error: %v", err),
		http.StatusInternalServerError,
	)
}

func ErrBadRequest(message string) *AppError {
	return NewAppError(
		ErrCodeBadRequest,
		message,
		http.StatusBadRequest,
	)
}