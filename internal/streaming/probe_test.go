package streaming

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestProbe(t *testing.T) {
	for _, codec := range []string{"H264", "H265"} {
		t.Run(codec, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				reader := bufio.NewReader(conn)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					method := strings.Fields(line)[0]
					var seq string
					for {
						header, err := reader.ReadString('\n')
						if err != nil {
							return
						}
						if header == "\r\n" {
							break
						}
						if strings.HasPrefix(strings.ToLower(header), "cseq:") {
							seq = strings.TrimSpace(strings.SplitN(header, ":", 2)[1])
						}
					}
					if method == "DESCRIBE" {
						sdp := "v=0\r\no=- 0 0 IN IP4 127.0.0.1\r\ns=Test\r\nc=IN IP4 127.0.0.1\r\nt=0 0\r\nm=video 0 RTP/AVP 96\r\na=rtpmap:96 " + codec + "/90000\r\na=control:trackID=0\r\n"
						fmt.Fprintf(conn, "RTSP/1.0 200 OK\r\nCSeq: %s\r\nContent-Type: application/sdp\r\nContent-Length: %d\r\n\r\n%s", seq, len(sdp), sdp)
					} else {
						fmt.Fprintf(conn, "RTSP/1.0 200 OK\r\nCSeq: %s\r\nPublic: OPTIONS, DESCRIBE\r\n\r\n", seq)
					}
				}
			}()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = Probe(ctx, "rtsp://"+listener.Addr().String()+"/video")
			if (err == nil) != (codec == "H264") {
				t.Fatalf("unexpected probe result: %v", err)
			}
		})
	}
}

func TestProbeTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 1024)
		for {
			if _, err := conn.Read(buf); err != nil {
				return
			}
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := Probe(ctx, "rtsp://user:SECRET@"+listener.Addr().String()+"/video"); err == nil {
		t.Fatal("expected timeout")
	}
	if time.Since(started) > time.Second {
		t.Fatal("probe exceeded deadline")
	}
}
