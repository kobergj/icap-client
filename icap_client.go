package icapclient

import (
	"errors"
	"fmt"
	"net/http/httputil"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// the icap request methods
const (
	MethodOPTIONS = "OPTIONS"
	MethodRESPMOD = "RESPMOD"
	MethodREQMOD  = "REQMOD"
)

// shared errors
var (
	// ErrNoContext is used when no context is provided
	ErrNoContext = errors.New("no context provided")

	// ErrInvalidScheme is used when the url scheme is not icap://
	ErrInvalidScheme = errors.New("the url scheme must be icap://")

	// ErrMethodNotAllowed is used when the method is not allowed
	ErrMethodNotAllowed = errors.New("the requested method is not registered")

	// ErrInvalidHost is used when the host is invalid
	ErrInvalidHost = errors.New("the requested host is invalid")

	// ErrInvalidTCPMsg is used when the tcp message is invalid
	ErrInvalidTCPMsg = errors.New("invalid tcp message")

	// ErrREQMODWithoutReq is used when the request is nil for REQMOD method
	ErrREQMODWithoutReq = errors.New("http request cannot be nil for method REQMOD")

	// ErrREQMODWithResp is used when the response is not nil for REQMOD method
	ErrREQMODWithResp = errors.New("http response must be nil for method REQMOD")

	// ErrRESPMODWithoutResp is used when the response is nil for RESPMOD method
	ErrRESPMODWithoutResp = errors.New("http response cannot be nil for method RESPMOD")
)

// general constants required for the package
const (
	schemeICAP                      = "icap"
	icapVersion                     = "ICAP/1.0"
	httpVersion                     = "HTTP/1.1"
	schemeHTTPReq                   = "http_request"
	schemeHTTPResp                  = "http_response"
	crlf                            = "\r\n"
	doubleCRLF                      = "\r\n\r\n"
	lf                              = "\n"
	bodyEndIndicator                = crlf + "0" + crlf
	fullBodyEndIndicatorPreviewMode = "; ieof" + doubleCRLF
	icap100ContinueMsg              = "ICAP/1.0 100 Continue" + doubleCRLF
	icap204NoModsMsg                = "ICAP/1.0 204 Unmodified"
	defaultTimeout                  = 15 * time.Second
)

// Common ICAP headers
const (
	previewHeader      = "Preview"
	encapsulatedHeader = "Encapsulated"
)

// Options holds the options for the icap client
type Options struct {
	// Timeout is the maximum amount of time a connection will be kept open
	Timeout time.Duration
}

// getStatusWithCode prepares the status code and status text from two given strings
func getStatusWithCode(str1, str2 string) (int, string, error) {

	statusCode, err := strconv.Atoi(str1)

	if err != nil {
		return 0, "", err
	}

	status := strings.TrimSpace(str2)

	return statusCode, status, nil
}

// getHeaderVal parses the header and its value from a tcp message string
func getHeaderVal(str string) (string, string) {

	headerVals := strings.SplitN(str, ":", 2)
	header := headerVals[0]
	val := ""

	if len(headerVals) >= 2 {
		val = strings.TrimSpace(headerVals[1])
	}

	return header, val

}

// isRequestLine determines if the tcp message string is a request line, i.e., the first line of the message or not
func isRequestLine(str string) bool {
	return strings.Contains(str, icapVersion) || strings.Contains(str, httpVersion)
}

// setEncapsulatedHeaderValue generates the Encapsulated values and assigns to the ICAP request string
func setEncapsulatedHeaderValue(icapReqStr *string, httpReqStr, httpRespStr string) {
	encpVal := " "

	if strings.HasPrefix(*icapReqStr, MethodOPTIONS) {
		if httpReqStr == "" && httpRespStr == "" {
			// the most common case for OPTIONS method, no Encapsulated body
			encpVal += "null-body=0"
		} else {
			// if there is an Encapsulated body
			encpVal += "opt-body=0"
		}
	}

	if strings.HasPrefix(*icapReqStr, MethodREQMOD) || strings.HasPrefix(*icapReqStr, MethodRESPMOD) {
		// looking for the match of the string \r\n\r\n,
		// as that is the expression that separates each block, i.e., headers and bodies
		re := regexp.MustCompile(doubleCRLF)

		// getting the offsets of the matches, tells us the starting/ending point of headers and bodies
		reqIndices := re.FindAllStringIndex(httpReqStr, -1)

		// is needed to calculate the response headers by adding the last offset of the request block
		reqEndsAt := 0

		if reqIndices != nil {
			encpVal += "req-hdr=0"
			reqEndsAt = reqIndices[0][1]

			// indicating there is a body present for the request block, as length would have been 1 for a single match of \r\n\r\n
			if len(reqIndices) > 1 {
				encpVal += fmt.Sprintf(", req-body=%d", reqIndices[0][1]) // assigning the starting point of the body
				reqEndsAt = reqIndices[1][1]
			} else if httpRespStr == "" {
				encpVal += fmt.Sprintf(", null-body=%d", reqIndices[0][1])
			}

			if httpRespStr != "" {
				encpVal += ", "
			}
		}

		respIndices := re.FindAllStringIndex(httpRespStr, -1)

		if respIndices != nil {
			encpVal += fmt.Sprintf("res-hdr=%d", reqEndsAt)
			if len(respIndices) > 1 {
				encpVal += fmt.Sprintf(", res-body=%d", reqEndsAt+respIndices[0][1])
			} else {
				encpVal += fmt.Sprintf(", null-body=%d", reqEndsAt+respIndices[0][1])
			}
		}

	}
	// formatting the ICAP request Encapsulated header with the value
	*icapReqStr = fmt.Sprintf(*icapReqStr, encpVal)
}

// replaceRequestURIWithActualURL replaces just the escaped portion of the url with the entire URL in the dumped request message
func replaceRequestURIWithActualURL(str *string, uri, url string) {
	if uri == "" {
		uri = "/"
	}
	*str = strings.Replace(*str, uri, url, 1)
}

// addFullBodyInPreviewIndicator adds 0; ieof\r\n\r\n which indicates the entire body fitted in the preview
func addFullBodyInPreviewIndicator(str *string) {
	*str = strings.TrimSuffix(*str, doubleCRLF)
	*str += fullBodyEndIndicatorPreviewMode
}

// splitBodyAndHeader separates header and body from a http message
func splitBodyAndHeader(str string) (string, string, bool) {
	ss := strings.SplitN(str, doubleCRLF, 2)

	if len(ss) < 2 || ss[1] == "" {
		return "", "", false
	}

	headerStr := ss[0]
	bodyStr := ss[1]

	return headerStr, bodyStr, true
}

// bodyAlreadyChunked determines if the http body is already chunked from the origin server or not
func bodyAlreadyChunked(str string) bool {
	_, bodyStr, ok := splitBodyAndHeader(str)

	if !ok {
		return false
	}

	r := regexp.MustCompile("\\r\\n0(\\r\\n)+$")
	return r.MatchString(bodyStr)

}

// parsePreviewBodyBytes parses the preview portion of the body and only keeps that in the message
func parsePreviewBodyBytes(str *string, pb int) {

	headerStr, bodyStr, ok := splitBodyAndHeader(*str)

	if !ok {
		return
	}

	bodyStr = bodyStr[:pb]

	*str = headerStr + doubleCRLF + bodyStr
}

// addHexBodyByteNotations adds the hexadecimal byte notations in the messages,
// for example, Hello World, becomes
// b
// Hello World
// 0
func addHexBodyByteNotations(bodyStr *string) {

	bodyBytes := []byte(*bodyStr)

	*bodyStr = fmt.Sprintf("%x%s%s%s", len(bodyBytes), crlf, *bodyStr, bodyEndIndicator)
}

// mergeHeaderAndBody merges the header and body of the http message
func mergeHeaderAndBody(src *string, headerStr, bodyStr string) {
	*src = headerStr + doubleCRLF + bodyStr
}

// toICAPMessage returns the given request in its ICAP/1.x wire
func toICAPMessage(req *Request) ([]byte, error) {

	// Making the ICAP message block
	reqStr := fmt.Sprintf("%s %s %s%s", req.Method, req.URL.String(), icapVersion, crlf)

	for headerName, vals := range req.Header {
		for _, val := range vals {
			reqStr += fmt.Sprintf("%s: %s%s", headerName, val, crlf)
		}
	}

	// will populate the Encapsulated header value after making the http Request & Response messages
	reqStr += "Encapsulated: %s" + crlf
	reqStr += crlf

	// build the HTTP Request message block
	httpReqStr := ""
	if req.HTTPRequest != nil {
		b, err := httputil.DumpRequestOut(req.HTTPRequest, true)

		if err != nil {
			return nil, err
		}

		httpReqStr += string(b)
		replaceRequestURIWithActualURL(&httpReqStr, req.HTTPRequest.URL.EscapedPath(), req.HTTPRequest.URL.String())

		if req.Method == MethodREQMOD {
			if req.previewSet {
				parsePreviewBodyBytes(&httpReqStr, req.PreviewBytes)
			}

			if !bodyAlreadyChunked(httpReqStr) {
				headerStr, bodyStr, ok := splitBodyAndHeader(httpReqStr)
				if ok {
					addHexBodyByteNotations(&bodyStr)
					mergeHeaderAndBody(&httpReqStr, headerStr, bodyStr)
				}
			}

		}

		// if the HTTP Request message block doesn't end with a \r\n\r\n,
		// then going to add one by force for better calculation of byte offsets
		if httpReqStr != "" {
			for !strings.HasSuffix(httpReqStr, doubleCRLF) {
				httpReqStr += crlf
			}
		}

	}

	// build the HTTP Response message block
	httpRespStr := ""
	if req.HTTPResponse != nil {
		b, err := httputil.DumpResponse(req.HTTPResponse, true)

		if err != nil {
			return nil, err
		}

		httpRespStr += string(b)

		if req.previewSet {
			parsePreviewBodyBytes(&httpRespStr, req.PreviewBytes)
		}

		if !bodyAlreadyChunked(httpRespStr) {
			headerStr, bodyStr, ok := splitBodyAndHeader(httpRespStr)
			if ok {
				addHexBodyByteNotations(&bodyStr)
				mergeHeaderAndBody(&httpRespStr, headerStr, bodyStr)
			}
		}

		if httpRespStr != "" && !strings.HasSuffix(httpRespStr, doubleCRLF) { // if the HTTP Response message block doesn't end with a \r\n\r\n, then going to add one by force for better calculation of byte offsets
			httpRespStr += crlf
		}

	}

	if encpVal := req.Header.Get(encapsulatedHeader); encpVal != "" {
		reqStr = fmt.Sprintf(reqStr, encpVal)
	} else {
		//populating the Encapsulated header of the ICAP message portion
		setEncapsulatedHeaderValue(&reqStr, httpReqStr, httpRespStr)
	}

	// determining if the http message needs the full body fitted in the preview portion indicator or not
	if httpRespStr != "" && req.previewSet && req.bodyFittedInPreview {
		addFullBodyInPreviewIndicator(&httpRespStr)
	}

	if req.Method == MethodREQMOD && req.previewSet && req.bodyFittedInPreview {
		addFullBodyInPreviewIndicator(&httpReqStr)
	}

	data := []byte(reqStr + httpReqStr + httpRespStr)

	return data, nil
}
