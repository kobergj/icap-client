package examples

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	ic "github.com/egirna/icap-client"
)

func makeReqmodCall() {
	ctx := context.Background()

	// prepare the http request
	httpReq, err := http.NewRequest(http.MethodGet, "http://localhost:8000/sample.pdf", nil)
	if err != nil {
		log.Fatal(err)
	}

	// create a new icap OPTIONS request
	optReq, err := ic.NewRequest(ctx, ic.MethodOPTIONS, "icap://127.0.0.1:1344/reqmod", nil, nil)
	if err != nil {
		log.Fatal(err)
	}

	// create the icap client
	client, err := ic.NewClient(ic.Options{
		Timeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	// send the OPTIONS request
	optResp, err := client.Do(optReq)
	if err != nil {
		log.Fatal(err)
	}

	// create a new icap REQMOD request
	req, err := ic.NewRequest(ctx, ic.MethodREQMOD, "icap://127.0.0.1:1344/reqmod", httpReq, nil)
	if err != nil {
		log.Fatal(err)
	}

	// set the preview bytes obtained from the OPTIONS call
	err = req.SetPreview(optResp.PreviewBytes)
	if err != nil {
		log.Fatal(err)
	}

	// send the REQMOD request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Status)
}
