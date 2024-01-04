package icapclient

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Response represents the icap server response data
type Response struct {
	StatusCode      int
	Status          string
	PreviewBytes    int
	Header          http.Header
	ContentRequest  *http.Request
	ContentResponse *http.Response
}

// readResponse reads the response from the icap server
func readResponse(b *bufio.Reader) (*Response, error) {

	resp := &Response{
		Header: make(map[string][]string),
	}

	scheme := ""
	httpMsg := ""
	for currentMsg, err := b.ReadString('\n'); err == nil || currentMsg != ""; currentMsg, err = b.ReadString('\n') { // keep reading the buffer message which is the http response message

		// if the current message line if the first line of the message portion(request line)
		if isRequestLine(currentMsg) {
			ss := strings.Split(currentMsg, " ")

			// must contain 3 words, for example, "ICAP/1.0 200 OK" or "GET /something HTTP/1.1"
			if len(ss) < 3 {
				return nil, fmt.Errorf("%w: %s", ErrInvalidTCPMsg, currentMsg)
			}

			// preparing the scheme below
			if ss[0] == icapVersion {
				scheme = schemeICAP
				resp.StatusCode, resp.Status, err = getStatusWithCode(ss[1], strings.Join(ss[2:], " "))
				if err != nil {
					return nil, err
				}
				continue
			}

			if ss[0] == httpVersion {
				scheme = schemeHTTPResp
				httpMsg = ""
			}

			// http request message scheme version should always be at the end,
			// for example, GET /something HTTP/1.1
			if strings.TrimSpace(ss[2]) == httpVersion {
				scheme = schemeHTTPReq
				httpMsg = ""
			}
		}

		// preparing the header for ICAP & contents for the HTTP messages below
		if scheme == schemeICAP {
			// ignore the CRLF and the LF, shouldn't be counted
			if currentMsg == lf || currentMsg == crlf {
				continue
			}

			header, val := getHeaderVal(currentMsg)
			if header == previewHeader {
				pb, _ := strconv.Atoi(val)
				resp.PreviewBytes = pb
			}

			resp.Header.Add(header, val)
		}

		if scheme == schemeHTTPReq {
			httpMsg += strings.TrimSpace(currentMsg) + crlf
			bufferEmpty := b.Buffered() == 0

			// a crlf indicates the end of the HTTP message and the buffer check is just in case the buffer ended with one last message instead of a crlf
			if currentMsg == crlf || bufferEmpty {
				var erR error
				resp.ContentRequest, erR = http.ReadRequest(bufio.NewReader(strings.NewReader(httpMsg)))
				if erR != nil {
					return nil, erR
				}
				continue
			}
		}

		if scheme == schemeHTTPResp {
			httpMsg += strings.TrimSpace(currentMsg) + crlf
			bufferEmpty := b.Buffered() == 0
			if currentMsg == crlf || bufferEmpty {
				var erR error
				resp.ContentResponse, erR = http.ReadResponse(bufio.NewReader(strings.NewReader(httpMsg)), resp.ContentRequest)
				if erR != nil {
					return nil, erR
				}
				continue
			}

		}

	}

	return resp, nil
}
