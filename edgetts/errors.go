package edgetts

import "errors"

var (
	// ErrUnknownResponse is raised when an unknown response is received from the server.
	ErrUnknownResponse = errors.New("edgetts: unknown response received from server")
	// ErrUnexpectedResponse is raised when an unexpected response is received from the server.
	ErrUnexpectedResponse = errors.New("edgetts: unexpected response received from server")
	// ErrNoAudioReceived is raised when no audio is received from the server.
	ErrNoAudioReceived = errors.New("edgetts: no audio was received; please verify that your parameters are correct")
	// ErrWebSocket is raised when a WebSocket error occurs.
	ErrWebSocket = errors.New("edgetts: websocket error")
	// ErrSkewAdjustment is raised when clock skew adjustment fails.
	ErrSkewAdjustment = errors.New("edgetts: clock skew adjustment failed")
)

// UnknownResponseError adds context to ErrUnknownResponse.
type UnknownResponseError struct{ Msg string }

func (e *UnknownResponseError) Error() string { return "edgetts: unknown response: " + e.Msg }
func (e *UnknownResponseError) Unwrap() error { return ErrUnknownResponse }

// UnexpectedResponseError adds context to ErrUnexpectedResponse.
type UnexpectedResponseError struct{ Msg string }

func (e *UnexpectedResponseError) Error() string { return "edgetts: unexpected response: " + e.Msg }
func (e *UnexpectedResponseError) Unwrap() error { return ErrUnexpectedResponse }

// NoAudioReceivedError adds context to ErrNoAudioReceived.
type NoAudioReceivedError struct{ Msg string }

func (e *NoAudioReceivedError) Error() string { return "edgetts: " + e.Msg }
func (e *NoAudioReceivedError) Unwrap() error { return ErrNoAudioReceived }

// WebSocketError adds context to ErrWebSocket.
type WebSocketError struct{ Msg string }

func (e *WebSocketError) Error() string { return "edgetts: websocket error: " + e.Msg }
func (e *WebSocketError) Unwrap() error { return ErrWebSocket }

// SkewAdjustmentError adds context to ErrSkewAdjustment.
type SkewAdjustmentError struct{ Msg string }

func (e *SkewAdjustmentError) Error() string { return "edgetts: skew adjustment: " + e.Msg }
func (e *SkewAdjustmentError) Unwrap() error { return ErrSkewAdjustment }
