package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var readInterceptorCh chan readInterceptor

type readInterceptor struct {
	reader        io.Reader
	id            int
	off           int64
	contentLength int64
}

func (ri *readInterceptor) Read(p []byte) (n int, err error) {
	n, err = ri.reader.Read(p)
	ri.off += int64(n)
	readInterceptorCh <- *ri
	return n, err
}

func main() {
	n := flag.Int("n", 1, "number of concurrent download")
	flag.Parse()

	if *n < 1 {
		*n = 1
	}

	m := make(map[int]readInterceptor, *n)
	for i := 0; i < *n; i++ {
		m[i+1] = readInterceptor{id: i + 1}
	}
	readInterceptorCh = make(chan readInterceptor)

	args := flag.Args()
	if len(args) == 0 {
		fatalf("missing url from arguments\n")
	}
	url := args[len(args)-1]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quitc := make(chan os.Signal, 1)
	signal.Notify(quitc, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT)
	go func() {
		<-quitc
		cancel()
	}()

	begin := time.Now()
	fmt.Println("Fetching metadata...")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fatalf("could not create request\n")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatalf("could not do request: %v\n", err)
	}
	defer resp.Body.Close()

	var filename string
	contentDisposition := resp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		urlParts := strings.Split(url, "/")
		filename = urlParts[len(urlParts)-1]
	}
	if params["filename"] != "" {
		filename = params["filename"]
	}

	acceptRanges := resp.Header.Get("Accept-Ranges")
	contentLength, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	fmt.Printf("Downloading %s: %s\n", filename, bytesFormatter(contentLength))

	printCh := make(chan struct{})
	go print(m, *n, begin, printCh)

	if acceptRanges == "" || contentLength <= 0 || *n == 1 {
		_, err = writeToFile(resp.Body, filename, 1, 0, contentLength)
		if err != nil {
			fatalf("could not write to file: %v\n", err)
		}
		fmt.Printf("\nCompleted in %s\n", time.Since(begin))
		return
	}

	var part int64
	remainder := contentLength % int64(*n)
	if remainder == 0 {
		part = contentLength / int64(*n)
	} else {
		part = (contentLength - remainder) / int64(*n)
	}

	var start int64 = 0
	var end int64 = part + remainder
	var wg sync.WaitGroup

	// keep the initial connection alive and download from it up to cutoff
	wg.Add(1)
	go func(wg *sync.WaitGroup, start, end int64) {
		defer wg.Done()

		_, err = writeToFile(resp.Body, filename, 1, start, end-start)
		if err != nil {
			fatalf("could not write to file: %v", err)
		}
	}(&wg, start, end)

	// partial remaining downloads
	start = end
	end = end + part
	for i := 1; i < *n; i++ {
		wg.Add(1)
		go download(ctx, &wg, i+1, url, filename, start, end)
		start = end
		end = end + part
	}
	wg.Wait()

	close(readInterceptorCh)
	<-printCh
}

func fatalf(formatter string, args ...interface{}) {
	fmt.Printf(formatter, args...)
	os.Exit(1)
}

func print(m map[int]readInterceptor, n int, begin time.Time, done chan<- struct{}) {
	var mu sync.Mutex

	fmt.Print("\033[s")
	for ch := range readInterceptorCh {
		mu.Lock()
		m[ch.id] = ch
		mu.Unlock()

		fmt.Print("\033[u\033[K")
		formatters := []string{}
		args := []interface{}{}
		for i := 1; i <= n; i++ {
			val := m[i]
			formatters = append(formatters, "[%d] %d/%d (%d%s)")
			var percentage int64
			if val.contentLength != 0 {
				percentage = val.off * 100 / val.contentLength
			}
			args = append(args, i, val.off, val.contentLength, percentage, "%")
		}
		fmt.Printf(strings.Join(formatters, "\n"), args...)
	}
	fmt.Printf("\nCompleted in %s\n", time.Since(begin))
	done <- struct{}{}
}

func download(ctx context.Context, wg *sync.WaitGroup, id int, url, filename string, start, end int64) {
	defer wg.Done()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fatalf("could not create request: %v", err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fatalf("could not do request: %v", err)
	}
	defer resp.Body.Close()

	contentLength, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if contentLength <= 0 {
		contentLength = end - start
	}

	_, err = writeToFile(resp.Body, filename, id, start, contentLength)
	if err != nil {
		fatalf("could not do request: %v", err)
	}
}

func writeToFile(reader io.Reader, filename string, id int, start, contentLength int64) (n int64, err error) {
	fout, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer fout.Close()

	_, err = fout.Seek(start, 0)
	if err != nil {
		return
	}

	n, err = io.CopyN(fout, &readInterceptor{id: id, reader: reader, contentLength: contentLength}, contentLength)
	if err != nil {
		fatalf("could not copy bytes of stream to file: %v", err)
	}
	return
}

func bytesFormatter(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}
