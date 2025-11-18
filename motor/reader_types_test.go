package motor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadRequestBuilder_Complete(t *testing.T) {
	buf := make([]byte, 100)

	req := NewReadRequestBuilder().
		WithOffset(1000).
		WithLength(500).
		WithBuffer(&buf).
		Build()

	assert.Equal(t, int64(1000), req.GetOffset())
	assert.Equal(t, int64(500), req.GetLength())
	assert.Equal(t, &buf, req.GetBuffer())
}

func TestReadRequestBuilder_WithoutBuffer(t *testing.T) {
	req := NewReadRequestBuilder().
		WithOffset(2000).
		WithLength(1000).
		Build()

	assert.Equal(t, int64(2000), req.GetOffset())
	assert.Equal(t, int64(1000), req.GetLength())
	assert.Nil(t, req.GetBuffer())
}

func TestReadRequestBuilder_MinimalFields(t *testing.T) {
	req := NewReadRequestBuilder().Build()

	assert.Equal(t, int64(0), req.GetOffset())
	assert.Equal(t, int64(0), req.GetLength())
	assert.Nil(t, req.GetBuffer())
}

func TestReadRequestBuilder_FluentAPI(t *testing.T) {
	// verify fluent api returns builder for chaining
	builder := NewReadRequestBuilder()
	b1 := builder.WithOffset(100)
	b2 := b1.WithLength(200)
	buf := make([]byte, 10)
	b3 := b2.WithBuffer(&buf)

	// all should return same builder type
	assert.IsType(t, builder, b1)
	assert.IsType(t, builder, b2)
	assert.IsType(t, builder, b3)
}

func TestReadResponse_GetEntry(t *testing.T) {
	resp := newReadResponse()
	assert.Nil(t, resp.GetEntry())

	// normally would have Entry from harhar but we can't construct one easily
	// so just verify nil initially
	assert.Equal(t, int64(0), resp.GetBytesRead())
	assert.Nil(t, resp.GetError())
}

func TestReadResponse_GetBytesRead(t *testing.T) {
	resp := newReadResponse()
	resp.bytesRead = 12345

	assert.Equal(t, int64(12345), resp.GetBytesRead())
}

func TestReadResponse_GetError(t *testing.T) {
	resp := newReadResponse()
	assert.Nil(t, resp.GetError())

	testErr := assert.AnError
	resp.err = testErr
	assert.Equal(t, testErr, resp.GetError())
}

func TestReadRequestBuilder_MultipleBuilds(t *testing.T) {
	// verify we can build multiple requests from same builder
	builder := NewReadRequestBuilder().
		WithOffset(100).
		WithLength(200)

	req1 := builder.Build()
	req2 := builder.Build()

	// both should have same values
	assert.Equal(t, req1.GetOffset(), req2.GetOffset())
	assert.Equal(t, req1.GetLength(), req2.GetLength())
}

func TestReadRequestBuilder_OverwriteValues(t *testing.T) {
	buf1 := make([]byte, 10)
	buf2 := make([]byte, 20)

	req := NewReadRequestBuilder().
		WithOffset(100).
		WithOffset(200). // overwrite
		WithLength(50).
		WithLength(75). // overwrite
		WithBuffer(&buf1).
		WithBuffer(&buf2). // overwrite
		Build()

	assert.Equal(t, int64(200), req.GetOffset())
	assert.Equal(t, int64(75), req.GetLength())
	assert.Equal(t, &buf2, req.GetBuffer())
}
