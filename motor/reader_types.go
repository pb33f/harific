package motor

import "github.com/pb33f/harhar"

// readRequest is the private implementation of ReadRequest
type readRequest struct {
	offset int64
	length int64
	buffer *[]byte
}

func (r *readRequest) GetOffset() int64   { return r.offset }
func (r *readRequest) GetLength() int64   { return r.length }
func (r *readRequest) GetBuffer() *[]byte { return r.buffer }

// readRequestBuilder is the private builder implementation
type readRequestBuilder struct {
	req readRequest
}

// NewReadRequestBuilder creates a new builder for constructing read requests
func NewReadRequestBuilder() ReadRequestBuilder {
	return &readRequestBuilder{}
}

func (b *readRequestBuilder) WithOffset(offset int64) ReadRequestBuilder {
	b.req.offset = offset
	return b
}

func (b *readRequestBuilder) WithLength(length int64) ReadRequestBuilder {
	b.req.length = length
	return b
}

func (b *readRequestBuilder) WithBuffer(buf *[]byte) ReadRequestBuilder {
	b.req.buffer = buf
	return b
}

func (b *readRequestBuilder) Build() ReadRequest {
	return &b.req
}

// readResponse is the private implementation of ReadResponse
type readResponse struct {
	entry     *harhar.Entry
	bytesRead int64
	err       error
}

func (r *readResponse) GetEntry() *harhar.Entry { return r.entry }
func (r *readResponse) GetBytesRead() int64     { return r.bytesRead }
func (r *readResponse) GetError() error         { return r.err }

// newReadResponse creates a new read response
func newReadResponse() *readResponse {
	return &readResponse{}
}
