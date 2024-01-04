package icapclient

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestRequest(t *testing.T) {
	t.Run("Request Factory", func(t *testing.T) {

		type testSample struct {
			urlStr    string
			reqMethod string
			httpReq   *http.Request
			httpResp  *http.Response
			err       error
		}

		sampleTable := []testSample{
			{
				urlStr:    "icap://localhost:1344/something",
				reqMethod: MethodOPTIONS,
				httpReq:   nil,
				httpResp:  nil,
				err:       nil,
			},
			{
				urlStr:    "icap://localhost:1344/something",
				reqMethod: MethodRESPMOD,
				httpReq:   nil,
				httpResp:  &http.Response{},
				err:       nil,
			},
			{
				urlStr:    "icap://localhost:1344/something",
				reqMethod: MethodREQMOD,
				httpReq:   &http.Request{},
				httpResp:  nil,
				err:       nil,
			},
			{
				urlStr:    "icap://localhost:1344/something",
				reqMethod: "invalid",
				httpReq:   nil,
				httpResp:  nil,
				err:       ErrMethodNotAllowed,
			},
			{
				urlStr:    "http://localhost:1344/something",
				reqMethod: MethodOPTIONS,
				httpReq:   nil,
				httpResp:  nil,
				err:       ErrInvalidScheme,
			},
			{
				urlStr:    "icap://",
				reqMethod: MethodOPTIONS,
				httpReq:   nil,
				httpResp:  nil,
				err:       ErrInvalidHost,
			},
			{
				urlStr:    "icap://localhost:1344/something",
				reqMethod: MethodREQMOD,
				httpReq:   nil,
				httpResp:  nil,
				err:       ErrREQMODWithoutReq,
			},
			{
				urlStr:    "icap://localhost:1344/something",
				reqMethod: MethodREQMOD,
				httpReq:   &http.Request{},
				httpResp:  &http.Response{},
				err:       ErrREQMODWithResp,
			},
			{
				urlStr:    "icap://localhost:1344/something",
				reqMethod: MethodRESPMOD,
				httpReq:   &http.Request{},
				httpResp:  nil,
				err:       ErrRESPMODWithoutResp,
			},
		}

		for _, sample := range sampleTable {
			if _, err := NewRequest(context.Background(), sample.reqMethod, sample.urlStr, sample.httpReq, sample.httpResp); !errors.Is(err, sample.err) {
				t.Logf("Wanted error: %v, got: %v", sample.err, err)
				t.Fail()
			}
		}

	})

	t.Run("setDefaultRequestHeaders", func(t *testing.T) {
		req, _ := NewRequest(context.Background(), MethodOPTIONS, "icap://localhost:1344/something", nil, nil)
		req.setDefaultRequestHeaders()

		if val, exists := req.Header["Allow"]; !exists || len(val) < 1 || val[0] != "204" {
			t.Log("Must have Allow header with 204 as value")
			t.Fail()
		}

		hname, _ := os.Hostname()
		if val, exists := req.Header["Host"]; !exists || len(val) < 1 || val[0] != hname {
			t.Logf("Must have Host header with %s as value", hname)
			t.Fail()
		}

		req, _ = NewRequest(context.Background(), MethodOPTIONS, "icap://localhost:1344/something", nil, nil)
		req.Header.Set("Host", "somehost")
		req.setDefaultRequestHeaders()

		if val, exists := req.Header["Host"]; !exists || len(val) < 1 || val[0] != "somehost" {
			t.Logf("Must have Host header with %s as value", "somehost")
			t.Fail()
		}

	})

	t.Run("extendHeader", func(t *testing.T) {
		type testSample struct {
			extendingHeader http.Header
			nameValue       []string
			addressValue    []string
			allowValue      []string
			defaultHeaders  bool
		}

		sampleTable := []testSample{
			{
				extendingHeader: http.Header{
					"Name":    []string{"some_name"},
					"Address": []string{"some_address1", "some_address2"},
					"Allow":   []string{"205"},
				},
				nameValue:      []string{"some_name"},
				addressValue:   []string{"some_address1", "some_address2"},
				allowValue:     []string{"205"},
				defaultHeaders: false,
			},
			{
				extendingHeader: http.Header{
					"Name":    []string{"some_name"},
					"Address": []string{"some_address1", "some_address2"},
					"Allow":   []string{"205"},
				},
				nameValue:      []string{"some_name"},
				addressValue:   []string{"some_address1", "some_address2"},
				allowValue:     []string{"204", "205"},
				defaultHeaders: true,
			},
		}

		for _, sample := range sampleTable {
			req, _ := NewRequest(context.Background(), MethodOPTIONS, "icap://localhost:1344/something", nil, nil)
			if sample.defaultHeaders {
				req.setDefaultRequestHeaders()
			}

			if err := req.extendHeader(sample.extendingHeader); err != nil {
				t.Fatal(err.Error())
			}

			if val, exists := req.Header["Allow"]; !exists || !reflect.DeepEqual(val, sample.allowValue) {
				t.Logf("Wanted Allow header with value: %v, got: %v", sample.allowValue, val)
				t.Fail()
			}

			if val, exists := req.Header["Name"]; !exists || !reflect.DeepEqual(val, sample.nameValue) {
				t.Logf("Wanted Name header with value: %v , got: %v", sample.nameValue, val)
				t.Fail()
			}

			if val, exists := req.Header["Address"]; !exists || !reflect.DeepEqual(val, sample.addressValue) {
				t.Logf("Wanted Address header with value: %v, got: %v", sample.addressValue, val)
				t.Fail()
			}

		}

	})

	t.Run("SetPreview", func(t *testing.T) {

		type testSample struct {
			reqMethod             string
			previewBytes          int
			bodyStr               string
			allocatedPreviewBytes int
			previewHeaderValue    []string
			remainingPreviewBytes []byte
			bodyFittedInPreview   bool
		}

		sampleTable := []testSample{
			{
				reqMethod:             MethodREQMOD,
				previewBytes:          11,
				bodyStr:               "Hello World! Bye Bye World!",
				allocatedPreviewBytes: 11,
				previewHeaderValue:    []string{"11"},
				remainingPreviewBytes: []byte(`! Bye Bye World!`),
				bodyFittedInPreview:   false,
			},
			{
				reqMethod:             MethodREQMOD,
				previewBytes:          11,
				bodyStr:               "Hello!",
				allocatedPreviewBytes: 6,
				previewHeaderValue:    []string{"6"},
				remainingPreviewBytes: nil,
				bodyFittedInPreview:   true,
			},
			{
				reqMethod:             MethodRESPMOD,
				previewBytes:          11,
				bodyStr:               "Hello World! Bye Bye World!",
				allocatedPreviewBytes: 11,
				previewHeaderValue:    []string{"11"},
				remainingPreviewBytes: []byte(`! Bye Bye World!`),
				bodyFittedInPreview:   false,
			},
			{
				reqMethod:             MethodRESPMOD,
				previewBytes:          11,
				bodyStr:               "Hello!",
				allocatedPreviewBytes: 6,
				previewHeaderValue:    []string{"6"},
				remainingPreviewBytes: nil,
				bodyFittedInPreview:   true,
			},
		}

		for _, sample := range sampleTable {
			bodyData := bytes.NewBufferString(sample.bodyStr)
			httpReq, _ := http.NewRequest(http.MethodPost, "http://someurl.com", bodyData)
			var req *Request
			if sample.reqMethod == MethodREQMOD {
				req, _ = NewRequest(context.Background(), sample.reqMethod, "icap://localhost:1344/something", httpReq, nil)
			}
			if sample.reqMethod == MethodRESPMOD {
				httpResp := &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Proto:      "HTTP/1.0",
					ProtoMajor: 1,
					ProtoMinor: 0,
					Header: http.Header{
						"Content-Type":   []string{"plain/text"},
						"Content-Length": []string{strconv.Itoa(bodyData.Len())},
					},
					ContentLength: int64(bodyData.Len()),
					Body:          ioutil.NopCloser(strings.NewReader(sample.bodyStr)),
				}
				req, _ = NewRequest(context.Background(), sample.reqMethod, "icap://localhost:1344/something", httpReq, httpResp)
			}

			if err := req.SetPreview(sample.previewBytes); err != nil {
				t.Fatal(err.Error())
			}

			if req.PreviewBytes != sample.allocatedPreviewBytes {
				t.Logf("Wanted preview bytes:%d, got:%d", sample.allocatedPreviewBytes, req.PreviewBytes)
				t.Fail()
			}

			var bdyBytes []byte

			if sample.reqMethod == MethodREQMOD {
				bdyBytes, _ = ioutil.ReadAll(req.HTTPRequest.Body)
			}

			if sample.reqMethod == MethodRESPMOD {
				bdyBytes, _ = ioutil.ReadAll(req.HTTPResponse.Body)
			}

			if string(bdyBytes) != sample.bodyStr {
				t.Logf("Wanted body string:%s, got:%s", sample.bodyStr, string(bdyBytes))
				t.Fail()
			}

			if val, exists := req.Header["Preview"]; !exists || !reflect.DeepEqual(val, sample.previewHeaderValue) {
				t.Logf("Wanted Preview header with value %v, got: %v", sample.previewHeaderValue, val)
				t.Fail()
			}

			if !reflect.DeepEqual(req.remainingPreviewBytes, sample.remainingPreviewBytes) {
				t.Logf("Wanted remaining preview bytes: %s, got: %s", string(sample.remainingPreviewBytes),
					string(req.remainingPreviewBytes))
				t.Fail()
			}

			if req.bodyFittedInPreview != sample.bodyFittedInPreview {
				t.Logf("Wanted body fitted in preview as: %v, got: %v", sample.bodyFittedInPreview, req.bodyFittedInPreview)
				t.Fail()
			}

		}

	})

}
