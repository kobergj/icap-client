package icapclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestClient_Do(t *testing.T) {
	if !testServerRunning() {
		go startTestServer()
	}

	t.Parallel()

	t.Run("RESPMOD", func(t *testing.T) {
		httpReq, err := http.NewRequest(http.MethodGet, "http://someurl.com", nil)
		if err != nil {
			t.Error(err)
			return
		}

		type testSample struct {
			httpResp         *http.Response
			wantedStatusCode int
			wantedStatus     string
		}

		sampleTable := []testSample{
			{
				httpResp: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Proto:      "HTTP/1.0",
					ProtoMajor: 1,
					ProtoMinor: 0,
					Header: http.Header{
						"Content-Type":   []string{"plain/text"},
						"Content-Length": []string{"19"},
					},
					ContentLength: 19,
					Body:          io.NopCloser(strings.NewReader("This is a GOOD FILE")),
				},
				wantedStatusCode: http.StatusNoContent,
				wantedStatus:     "No Modifications",
			},
			{
				httpResp: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Proto:      "HTTP/1.0",
					ProtoMajor: 1,
					ProtoMinor: 0,
					Header: http.Header{
						"Content-Type":   []string{"plain/text"},
						"Content-Length": []string{"18"},
					},
					ContentLength: 18,
					Body:          io.NopCloser(strings.NewReader("This is a BAD FILE")),
				},
				wantedStatusCode: http.StatusOK,
				wantedStatus:     "OK",
			},
		}

		for _, sample := range sampleTable {
			req, err := NewRequest(context.Background(), MethodRESPMOD, fmt.Sprintf("icap://localhost:%d/respmod", port), httpReq, sample.httpResp)
			if err != nil {
				t.Error(err)
				return
			}

			client, _ := NewClient()
			resp, err := client.Do(req)
			if err != nil {
				t.Error(err)
				return
			}

			if resp.StatusCode != sample.wantedStatusCode {
				t.Errorf("Wanted status code:%d, got:%d", sample.wantedStatusCode, resp.StatusCode)
			}

			if resp.Status != sample.wantedStatus {
				t.Errorf("Wanted status:%s, got:%s", sample.wantedStatus, resp.Status)
			}
		}

	})

	t.Run("REQMOD", func(t *testing.T) {
		type testSample struct {
			urlStr           string
			wantedStatusCode int
			wantedStatus     string
		}

		sampleTable := []testSample{
			{
				urlStr:           "http://goodifle.com",
				wantedStatusCode: http.StatusNoContent,
				wantedStatus:     "No Modifications",
			},
			{
				urlStr:           "http://badfile.com",
				wantedStatusCode: http.StatusOK,
				wantedStatus:     "OK",
			},
		}

		for _, sample := range sampleTable {
			httpReq, err := http.NewRequest(http.MethodGet, sample.urlStr, nil)
			if err != nil {
				t.Error(err)
				return
			}

			req, err := NewRequest(context.Background(), MethodREQMOD, fmt.Sprintf("icap://localhost:%d/reqmod", port), httpReq, nil)
			if err != nil {
				t.Error(err)
				return
			}

			client, _ := NewClient()
			resp, err := client.Do(req)
			if err != nil {
				t.Error(err)
				return
			}

			if resp.StatusCode != sample.wantedStatusCode {
				t.Errorf("Wanted status code:%d, got:%d", sample.wantedStatusCode, resp.StatusCode)
			}

			if resp.Status != sample.wantedStatus {
				t.Errorf("Wanted status:%s, got:%s", sample.wantedStatus, resp.Status)
			}
		}
	})

	t.Run("RESPMOD with OPTIONS", func(t *testing.T) {
		httpReq, err := http.NewRequest(http.MethodGet, "http://someurl.com", nil)
		if err != nil {
			t.Error(err)
			return
		}

		type testSample struct {
			httpResp               *http.Response
			wantedStatusCode       int
			wantedStatus           string
			wantedPreviewBytes     int
			wantedOptionStatusCode int
			wantedOptionStatus     string
			wantedOptionHeader     http.Header
		}

		sampleTable := []testSample{
			{
				httpResp: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Proto:      "HTTP/1.0",
					ProtoMajor: 1,
					ProtoMinor: 0,
					Header: http.Header{
						"Content-Type":   []string{"plain/text"},
						"Content-Length": []string{"41"},
					},
					ContentLength: 41,
					Body:          io.NopCloser(strings.NewReader("Hello World!This is a GOOD FILE! bye bye!")),
				},
				wantedStatusCode:       http.StatusNoContent,
				wantedStatus:           "No Modifications",
				wantedPreviewBytes:     previewBytes,
				wantedOptionStatusCode: http.StatusOK,
				wantedOptionStatus:     "OK",
				wantedOptionHeader: http.Header{
					"Methods":          []string{"RESPMOD"},
					"Allow":            []string{"204"},
					"Preview":          []string{strconv.Itoa(previewBytes)},
					"Transfer-Preview": []string{"*"},
				},
			},
			{
				httpResp: &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Proto:      "HTTP/1.0",
					ProtoMajor: 1,
					ProtoMinor: 0,
					Header: http.Header{
						"Content-Type":   []string{"plain/text"},
						"Content-Length": []string{"18"},
					},
					ContentLength: 18,
					Body:          io.NopCloser(strings.NewReader("This is a BAD FILE")),
				},
				wantedStatusCode:       http.StatusOK,
				wantedStatus:           "OK",
				wantedPreviewBytes:     previewBytes,
				wantedOptionStatusCode: http.StatusOK,
				wantedOptionStatus:     "OK",
				wantedOptionHeader: http.Header{
					"Methods":          []string{"RESPMOD"},
					"Allow":            []string{"204"},
					"Preview":          []string{strconv.Itoa(previewBytes)},
					"Transfer-Preview": []string{"*"},
				},
			},
		}

		for _, sample := range sampleTable {
			urlStr := fmt.Sprintf("icap://localhost:%d/respmod", port)

			optReq, err := NewRequest(context.Background(), MethodOPTIONS, urlStr, nil, nil)
			if err != nil {
				t.Error(err)
				return
			}

			client, _ := NewClient()
			optResp, err := client.Do(optReq)
			if err != nil {
				t.Error(err)
				return
			}

			if optResp.Status != sample.wantedOptionStatus {
				t.Errorf("Wanted status:%s, got:%s", sample.wantedOptionStatus, optResp.Status)
			}

			if optResp.StatusCode != sample.wantedOptionStatusCode {
				t.Errorf("Wanted status code:%d, got:%d", sample.wantedOptionStatusCode, optResp.StatusCode)
			}

			if optResp.PreviewBytes != sample.wantedPreviewBytes {
				t.Errorf("Wanted preview bytes:%d , got:%d", sample.wantedPreviewBytes, optResp.PreviewBytes)
			}

			for k, v := range sample.wantedOptionHeader {
				if val, exists := optResp.Header[k]; exists {
					if !reflect.DeepEqual(val, v) {
						t.Errorf("Wanted value for header:%s to be:%v, got:%v", k, v, val)
					}
					continue
				}

				t.Errorf("Expected header:%s but not found", k)
			}

			req, err := NewRequest(context.Background(), MethodRESPMOD, urlStr, httpReq, sample.httpResp)
			if err != nil {
				t.Error(err)
				return
			}

			if err := req.extendHeader(optResp.Header); err != nil {
				t.Error(err)
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Error(err)
				return
			}

			if resp.StatusCode != sample.wantedStatusCode {
				t.Errorf("Wanted status code:%d, got:%d", sample.wantedStatusCode, resp.StatusCode)
			}

			if resp.Status != sample.wantedStatus {
				t.Errorf("Wanted status:%s, got:%s", sample.wantedStatus, resp.Status)
			}

		}
	})

	t.Run("REQMOD with OPTIONS", func(t *testing.T) {
		type testSample struct {
			urlStr                 string
			wantedStatusCode       int
			wantedStatus           string
			wantedOptionStatus     string
			wantedOptionStatusCode int
			wantedOptionHeader     http.Header
		}

		sampleTable := []testSample{
			{
				urlStr:                 "http://goodifle.com",
				wantedStatusCode:       http.StatusNoContent,
				wantedStatus:           "No Modifications",
				wantedOptionStatus:     "OK",
				wantedOptionStatusCode: http.StatusOK,
				wantedOptionHeader: http.Header{
					"Methods":          []string{"REQMOD"},
					"Allow":            []string{"204"},
					"Preview":          []string{strconv.Itoa(previewBytes)},
					"Transfer-Preview": []string{"*"},
				},
			},
			{
				urlStr:                 "http://badfile.com",
				wantedStatusCode:       http.StatusOK,
				wantedStatus:           "OK",
				wantedOptionStatus:     "OK",
				wantedOptionStatusCode: http.StatusOK,
				wantedOptionHeader: http.Header{
					"Methods":          []string{"REQMOD"},
					"Allow":            []string{"204"},
					"Preview":          []string{strconv.Itoa(previewBytes)},
					"Transfer-Preview": []string{"*"},
				},
			},
		}

		for _, sample := range sampleTable {

			urlStr := fmt.Sprintf("icap://localhost:%d/reqmod", port)

			optReq, err := NewRequest(context.Background(), MethodOPTIONS, urlStr, nil, nil)
			if err != nil {
				t.Error(err)
				return
			}

			client, _ := NewClient()
			optResp, err := client.Do(optReq)
			if err != nil {
				t.Error(err)
				return
			}

			if optResp.Status != sample.wantedOptionStatus {
				t.Errorf("Wanted status:%s , got:%s", sample.wantedOptionStatus, optResp.Status)
			}
			if optResp.StatusCode != sample.wantedOptionStatusCode {
				t.Errorf("Wanted status code:%d , got:%d", sample.wantedOptionStatusCode, optResp.StatusCode)
			}
			for k, v := range sample.wantedOptionHeader {
				if val, exists := optResp.Header[k]; exists {
					if !reflect.DeepEqual(val, v) {
						t.Errorf("Wanted header:%s to have value:%v, got:%v", k, v, val)
					}
					continue
				}

				t.Errorf("Expected header:%s but not found", k)
			}

			httpReq, err := http.NewRequest(http.MethodGet, sample.urlStr, nil)
			if err != nil {
				t.Error(err)
				return
			}

			req, err := NewRequest(context.Background(), MethodREQMOD, urlStr, httpReq, nil)
			if err != nil {
				t.Error(err)
				return
			}

			if err := req.extendHeader(optResp.Header); err != nil {
				t.Error(err)
				return
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Error(err)
				return
			}

			if resp.StatusCode != sample.wantedStatusCode {
				t.Errorf("Wanted status code:%d, got:%d", sample.wantedStatusCode, resp.StatusCode)
			}

			if resp.Status != sample.wantedStatus {
				t.Errorf("Wanted status:%s, got:%s", sample.wantedStatus, resp.Status)
			}

		}
	})

	t.Run("Client Do REQMOD with Custom Driver", func(t *testing.T) {

		type testSample struct {
			urlStr           string
			wantedStatusCode int
			wantedStatus     string
		}

		sampleTable := []testSample{
			{
				urlStr:           "http://goodifle.com",
				wantedStatusCode: http.StatusNoContent,
				wantedStatus:     "No Modifications",
			},
			{
				urlStr:           "http://badfile.com",
				wantedStatusCode: http.StatusOK,
				wantedStatus:     "OK",
			},
		}

		for _, sample := range sampleTable {
			httpReq, err := http.NewRequest(http.MethodGet, sample.urlStr, nil)
			if err != nil {
				t.Error(err)
				return
			}

			req, err := NewRequest(context.Background(), MethodREQMOD, fmt.Sprintf("icap://localhost:%d/reqmod", port), httpReq, nil)
			if err != nil {
				t.Error(err)
				return
			}

			client, _ := NewClient()
			resp, err := client.Do(req)
			if err != nil {
				t.Error(err)
				return
			}

			if resp.StatusCode != sample.wantedStatusCode {
				t.Errorf("Wanted status code:%d, got:%d", sample.wantedStatusCode, resp.StatusCode)
			}

			if resp.Status != sample.wantedStatus {
				t.Errorf("Wanted status:%s, got:%s", sample.wantedStatus, resp.Status)
			}

		}
	})

	if testServerRunning() {
		defer stopTestServer()
	}
}
