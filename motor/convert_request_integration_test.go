package motor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/harific/motor/model"
	"github.com/pb33f/testify/require"
)

func TestStreamHARConvertRequestPreservesQuery(t *testing.T) {
	// Exercise the streaming/conversion path used by Wiretap's HAR import.
	const har = `{"log":{"version":"1.2","creator":{"name":"test","version":"1"},"entries":[{
		"startedDateTime":"2026-08-24T12:00:00Z","time":1,
		"request":{"method":"GET","url":"https://example.com/?tag=a&tag=b&empty=","httpVersion":"HTTP/1.1",
		"queryString":[{"name":"tag","value":"a"},{"name":"tag","value":"b"},{"name":"empty","value":""}],
		"headers":[],"cookies":[],"headersSize":-1,"bodySize":0},
		"response":{"status":200,"statusText":"OK","httpVersion":"HTTP/1.1","headers":[],"cookies":[],
		"content":{"size":0,"mimeType":"text/plain"},"redirectURL":"","headersSize":-1,"bodySize":0},
		"cache":{},"timings":{"send":0,"wait":1,"receive":0}
	}]}}`
	path := filepath.Join(t.TempDir(), "query.har")
	require.NoError(t, os.WriteFile(path, []byte(har), 0600))
	streamer, err := NewHARStreamer(path, StreamerOptions{WorkerCount: 1})
	require.NoError(t, err)
	defer streamer.Close()
	ctx := context.Background()
	require.NoError(t, streamer.Initialize(ctx))
	results, err := streamer.StreamRange(ctx, 0, streamer.GetIndex().TotalEntries)
	require.NoError(t, err)
	count := 0
	for result := range results {
		require.NoError(t, result.Error)
		require.NotNil(t, result.Entry)
		req, err := model.ConvertRequestIntoHttpRequest(result.Entry.Request)
		require.NoError(t, err)
		req.Body.Close()
		require.Equal(t, "https://example.com/?tag=a&tag=b&empty=", req.URL.String())
		require.Equal(t, []string{"a", "b"}, req.URL.Query()["tag"])
		require.Equal(t, []string{""}, req.URL.Query()["empty"])
		count++
	}
	require.Equal(t, 1, count)
}
