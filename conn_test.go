package icapclient_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/phayes/freeport"

	icapclient "github.com/egirna/icap-client"
)

func TestICAPConn_Send(t *testing.T) {
	port, err := freeport.GetFreePort()
	if err != nil {
		t.Fatal(err)
	}

	tcp, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer tcp.Close()

	clientConn, err := icapclient.NewICAPConn(icapclient.ICAPConnConfig{Timeout: 50 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	err = clientConn.Connect(context.Background(), tcp.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()

	tcpConn, err := tcp.Accept()
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}
	defer tcpConn.Close()

	{ // test section: send a request to the tcp // icap server
		tests := []struct {
			name      string
			messages  []string
			want      string
			terminate bool
		}{
			{
				name:     "icap100ContinueMsg",
				messages: []string{icapclient.ICAP100ContinueMsg},
				want:     icapclient.ICAP100ContinueMsg,
			},
			{
				name:     "doubleCRLF",
				messages: []string{"prefix" + icapclient.DoubleCRLF},
				want:     "prefix" + icapclient.DoubleCRLF,
			},
			{
				name:     "icap204NoModsMsg",
				messages: []string{"prefix" + icapclient.ICAP204NoModsMsg + "suffix"},
				want:     "prefix" + icapclient.ICAP204NoModsMsg + "suffix",
			},
		}

		for _, tc := range tests {
			t.Run(fmt.Sprintf("send/receive message: %s", tc.name), func(t *testing.T) {
				for _, message := range tc.messages {
					_, err = tcpConn.Write([]byte(message))
					if err != nil {
						t.Fatal(err)
					}
				}

				res, err := clientConn.Send(nil)
				if err != nil {
					t.Fatal(err)
				}

				if got := string(res); got != tc.want {
					t.Errorf("ICAPConn.Send() = %v, want %v", got, tc.want)
				}
			})
		}
	}
}
